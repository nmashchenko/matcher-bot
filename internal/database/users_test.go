package database

import (
	"context"
	"database/sql"
	"os"
	"testing"

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

	t.Cleanup(func() {
		db.NewDelete().TableExpr("users").Where("telegram_id >= 900000").Exec(context.Background())
		db.Close()
	})

	return db
}

func TestFindOrCreate_New(t *testing.T) {
	db := setupTestDB(t)
	users := NewUserStore(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900001).Exec(ctx)

	username := "testuser"
	firstName := "Test"
	user, err := users.FindOrCreate(ctx, 900001, &username, &firstName, nil)
	if err != nil {
		t.Fatalf("FindOrCreate: %v", err)
	}
	if user.TelegramID != 900001 {
		t.Errorf("TelegramID = %d; want 900001", user.TelegramID)
	}
}

func TestFindOrCreate_Existing(t *testing.T) {
	db := setupTestDB(t)
	users := NewUserStore(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900002).Exec(ctx)

	username := "user1"
	firstName := "First"
	users.FindOrCreate(ctx, 900002, &username, &firstName, nil)

	newUsername := "user1_updated"
	user, err := users.FindOrCreate(ctx, 900002, &newUsername, &firstName, nil)
	if err != nil {
		t.Fatalf("FindOrCreate (update): %v", err)
	}
	if user.Username == nil || *user.Username != "user1_updated" {
		t.Errorf("Username not updated")
	}
}

func TestGetByTelegramID(t *testing.T) {
	db := setupTestDB(t)
	users := NewUserStore(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900003).Exec(ctx)

	username := "gettest"
	users.FindOrCreate(ctx, 900003, &username, nil, nil)

	user, err := users.GetByTelegramID(ctx, 900003)
	if err != nil {
		t.Fatalf("GetByTelegramID: %v", err)
	}
	if user.TelegramID != 900003 {
		t.Errorf("TelegramID = %d; want 900003", user.TelegramID)
	}
}

func TestUpdate_Age(t *testing.T) {
	db := setupTestDB(t)
	users := NewUserStore(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900010).Exec(ctx)
	username := "agetest"
	users.FindOrCreate(ctx, 900010, &username, nil, nil)

	age := 25
	err := users.Update(ctx, 900010, &UserUpdateData{Age: &age})
	if err != nil {
		t.Fatalf("Update age: %v", err)
	}

	user, err := users.GetByTelegramID(ctx, 900010)
	if err != nil {
		t.Fatalf("GetByTelegramID: %v", err)
	}
	if user.Age == nil || *user.Age != 25 {
		t.Errorf("expected age 25, got %v", user.Age)
	}
}

func TestUpdate_UserState(t *testing.T) {
	db := setupTestDB(t)
	users := NewUserStore(db)
	ctx := context.Background()

	db.NewDelete().TableExpr("users").Where("telegram_id = ?", 900012).Exec(ctx)
	username := "statetest"
	users.FindOrCreate(ctx, 900012, &username, nil, nil)

	state := StateOnboarding
	err := users.Update(ctx, 900012, &UserUpdateData{UserState: &state})
	if err != nil {
		t.Fatalf("Update state: %v", err)
	}

	user, err := users.GetByTelegramID(ctx, 900012)
	if err != nil {
		t.Fatalf("GetByTelegramID: %v", err)
	}
	if user.UserState != StateOnboarding {
		t.Errorf("expected state %s, got %s", StateOnboarding, user.UserState)
	}
}
