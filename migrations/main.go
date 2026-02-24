package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type migration struct {
	version string
	up      func(context.Context, *bun.DB) error
	down    func(context.Context, *bun.DB) error
}

var migrations = []migration{
	{version: "001", up: up001, down: down001},
	{version: "002", up: up002, down: down002},
}

func ensureMigrationsTable(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp
		)
	`)
	return err
}

func isApplied(ctx context.Context, db *bun.DB, version string) bool {
	var count int
	err := db.NewSelect().
		TableExpr("schema_migrations").
		ColumnExpr("COUNT(*)").
		Where("version = ?", version).
		Scan(ctx, &count)
	return err == nil && count > 0
}

func markApplied(ctx context.Context, db *bun.DB, version string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", version)
	return err
}

func markUnapplied(ctx context.Context, db *bun.DB, version string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = ?", version)
	return err
}

func runUp(ctx context.Context, db *bun.DB) error {
	for _, m := range migrations {
		if isApplied(ctx, db, m.version) {
			fmt.Printf("  %s: already applied, skipping\n", m.version)
			continue
		}
		fmt.Printf("  %s: applying...\n", m.version)
		if err := m.up(ctx, db); err != nil {
			return fmt.Errorf("migration %s up: %w", m.version, err)
		}
		if err := markApplied(ctx, db, m.version); err != nil {
			return fmt.Errorf("mark %s applied: %w", m.version, err)
		}
		fmt.Printf("  %s: done\n", m.version)
	}
	return nil
}

func runDown(ctx context.Context, db *bun.DB) error {
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		if !isApplied(ctx, db, m.version) {
			fmt.Printf("  %s: not applied, skipping\n", m.version)
			continue
		}
		fmt.Printf("  %s: rolling back...\n", m.version)
		if err := m.down(ctx, db); err != nil {
			return fmt.Errorf("migration %s down: %w", m.version, err)
		}
		if err := markUnapplied(ctx, db, m.version); err != nil {
			return fmt.Errorf("unmark %s: %w", m.version, err)
		}
		fmt.Printf("  %s: done\n", m.version)
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./migrations <up|down|reset|status>")
		os.Exit(1)
	}

	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	ctx := context.Background()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		log.Fatalf("ensure migrations table: %v", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "up":
		fmt.Println("Running migrations up...")
		if err := runUp(ctx, db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Migrations applied successfully.")

	case "down":
		fmt.Println("Running migrations down...")
		if err := runDown(ctx, db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Migrations rolled back successfully.")

	case "reset":
		fmt.Println("Resetting database...")
		if _, err := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
			log.Fatalf("drop schema: %v", err)
		}
		if err := ensureMigrationsTable(ctx, db); err != nil {
			log.Fatalf("ensure migrations table: %v", err)
		}
		if err := runUp(ctx, db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Database reset successfully.")

	case "status":
		fmt.Println("Migration status:")
		for _, m := range migrations {
			applied := isApplied(ctx, db, m.version)
			status := "pending"
			if applied {
				status = "applied"
			}
			fmt.Printf("  %s: %s\n", m.version, status)
		}

	default:
		fmt.Printf("Unknown command: %s\nUsage: go run ./migrations <up|down|reset|status>\n", cmd)
		os.Exit(1)
	}
}
