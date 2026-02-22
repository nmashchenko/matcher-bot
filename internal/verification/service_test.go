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

func TestFindOrCreateUser_New(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Cleanup before test
	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900001).Exec(ctx)

	username := "testuser"
	firstName := "Test"
	user, err := svc.FindOrCreateUser(ctx, 900001, &username, &firstName, nil)
	if err != nil {
		t.Fatalf("FindOrCreateUser: %v", err)
	}
	if user.TelegramID != 900001 {
		t.Errorf("TelegramID = %d; want 900001", user.TelegramID)
	}
}

func TestFindOrCreateUser_Existing(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900002).Exec(ctx)

	username := "user1"
	firstName := "First"
	svc.FindOrCreateUser(ctx, 900002, &username, &firstName, nil)

	// Update with new info
	newUsername := "user1_updated"
	user, err := svc.FindOrCreateUser(ctx, 900002, &newUsername, &firstName, nil)
	if err != nil {
		t.Fatalf("FindOrCreateUser (update): %v", err)
	}
	if user.Username == nil || *user.Username != "user1_updated" {
		t.Errorf("Username not updated")
	}
}

func TestGetVerificationStatus(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900004).Exec(ctx)

	username := "status"
	svc.FindOrCreateUser(ctx, 900004, &username, nil, nil)

	status, err := svc.GetVerificationStatus(ctx, 900004)
	if err != nil {
		t.Fatalf("GetVerificationStatus: %v", err)
	}
	if status.Status != database.StatusPending {
		t.Errorf("Status = %q; want PENDING", status.Status)
	}
}
