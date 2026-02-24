package main

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func up002(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE EXTENSION IF NOT EXISTS vector;
		ALTER TABLE users
			ADD COLUMN IF NOT EXISTS avatar_file_id          TEXT,
			ADD COLUMN IF NOT EXISTS age                     INTEGER,
			ADD COLUMN IF NOT EXISTS goal                    TEXT,
			ADD COLUMN IF NOT EXISTS bio                     TEXT,
			ADD COLUMN IF NOT EXISTS looking_for             TEXT,
			ADD COLUMN IF NOT EXISTS bio_embedding           vector(1536),
			ADD COLUMN IF NOT EXISTS looking_for_embedding   vector(1536);
		CREATE INDEX IF NOT EXISTS idx_users_bio_embedding ON users USING hnsw (bio_embedding vector_cosine_ops);
		CREATE INDEX IF NOT EXISTS idx_users_looking_for_embedding ON users USING hnsw (looking_for_embedding vector_cosine_ops);
	`)
	if err != nil {
		return fmt.Errorf("migration 002 up: %w", err)
	}
	return nil
}

func down002(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		DROP INDEX IF EXISTS idx_users_looking_for_embedding;
		DROP INDEX IF EXISTS idx_users_bio_embedding;
		ALTER TABLE users
			DROP COLUMN IF EXISTS avatar_file_id,
			DROP COLUMN IF EXISTS age,
			DROP COLUMN IF EXISTS goal,
			DROP COLUMN IF EXISTS bio,
			DROP COLUMN IF EXISTS looking_for,
			DROP COLUMN IF EXISTS bio_embedding,
			DROP COLUMN IF EXISTS looking_for_embedding;
		DROP EXTENSION IF EXISTS vector;
	`)
	if err != nil {
		return fmt.Errorf("migration 002 down: %w", err)
	}
	return nil
}
