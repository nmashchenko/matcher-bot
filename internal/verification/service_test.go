package verification

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"matcher-bot/internal/database"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func setupTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("database not reachable: %v", err)
	}

	// Clean up test data
	t.Cleanup(func() {
		db.NewDelete().TableExpr("users").Where("telegram_id >= 900000").Exec(context.Background())
		db.Close()
	})

	return db
}

func TestGetVerificationStatus(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	users := database.NewUserStore(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900004).Exec(ctx)

	username := "status"
	users.FindOrCreate(ctx, 900004, &username, nil, nil)

	status, err := svc.GetVerificationStatus(ctx, 900004)
	if err != nil {
		t.Fatalf("GetVerificationStatus: %v", err)
	}
	if status.Status != database.StatusPending {
		t.Errorf("Status = %q; want PENDING", status.Status)
	}
}
