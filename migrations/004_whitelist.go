package main

import (
	"context"

	"github.com/uptrace/bun"
)

func up004(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE whitelist (
			telegram_id BIGINT PRIMARY KEY
		);
		INSERT INTO whitelist (telegram_id) VALUES (454332651);
	`)
	return err
}

func down004(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS whitelist`)
	return err
}
