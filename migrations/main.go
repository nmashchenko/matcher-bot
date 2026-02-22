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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./migrations <up|down|reset>")
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
	cmd := os.Args[1]

	switch cmd {
	case "up":
		fmt.Println("Running migrations up...")
		if err := up001(ctx, db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Migrations applied successfully.")

	case "down":
		fmt.Println("Running migrations down...")
		if err := down001(ctx, db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Migrations rolled back successfully.")

	case "reset":
		fmt.Println("Resetting database...")
		if err := down001(ctx, db); err != nil {
			log.Fatal(err)
		}
		if err := up001(ctx, db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Database reset successfully.")

	default:
		fmt.Printf("Unknown command: %s\nUsage: go run ./migrations <up|down|reset>\n", cmd)
		os.Exit(1)
	}
}
