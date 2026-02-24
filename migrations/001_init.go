package main

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func up001(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			telegram_id         BIGINT UNIQUE NOT NULL,
			username            TEXT,
			first_name          TEXT,
			last_name           TEXT,
			user_state          TEXT NOT NULL DEFAULT 'unverified',
			latitude            DOUBLE PRECISION,
			longitude           DOUBLE PRECISION,
			country             TEXT,
			state               TEXT,
			city                TEXT,
			verified_at         TIMESTAMPTZ,
			created_at          TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
			updated_at          TIMESTAMPTZ NOT NULL DEFAULT current_timestamp
		);
	`)
	if err != nil {
		return fmt.Errorf("migration 001 up: %w", err)
	}
	return nil
}

func down001(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		DROP TABLE IF EXISTS users;
	`)
	if err != nil {
		return fmt.Errorf("migration 001 down: %w", err)
	}
	return nil
}
