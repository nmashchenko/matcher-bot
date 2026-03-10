package main

import (
	"context"

	"github.com/uptrace/bun"
)

func up005(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE events ADD COLUMN IF NOT EXISTS min_age INT;
		ALTER TABLE events ADD COLUMN IF NOT EXISTS max_age INT;
	`)
	return err
}

func down005(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE events DROP COLUMN IF EXISTS min_age;
		ALTER TABLE events DROP COLUMN IF EXISTS max_age;
	`)
	return err
}
