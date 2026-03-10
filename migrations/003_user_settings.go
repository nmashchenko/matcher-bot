package main

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func up003(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS preferred_event_type TEXT;
	`)
	if err != nil {
		return fmt.Errorf("migration 003 up: %w", err)
	}
	return nil
}

func down003(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE users DROP COLUMN IF EXISTS preferred_event_type;
	`)
	if err != nil {
		return fmt.Errorf("migration 003 down: %w", err)
	}
	return nil
}
