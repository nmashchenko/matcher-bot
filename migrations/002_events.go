package main

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func up002(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS events (
			id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			host_telegram_id BIGINT NOT NULL REFERENCES users(telegram_id),
			title            TEXT NOT NULL,
			description      TEXT,
			event_type       TEXT NOT NULL,
			event_state      TEXT NOT NULL DEFAULT 'active',
			latitude         DOUBLE PRECISION NOT NULL,
			longitude        DOUBLE PRECISION NOT NULL,
			city             TEXT NOT NULL,
			state            TEXT NOT NULL,
			max_participants INTEGER NOT NULL DEFAULT 10,
			starts_at        TIMESTAMPTZ NOT NULL,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
			updated_at       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp
		);
		CREATE INDEX IF NOT EXISTS idx_events_state ON events(event_state);
		CREATE INDEX IF NOT EXISTS idx_events_starts_at ON events(starts_at);
		CREATE INDEX IF NOT EXISTS idx_events_host ON events(host_telegram_id);
		CREATE INDEX IF NOT EXISTS idx_events_city_state ON events(city, state);

		CREATE TABLE IF NOT EXISTS event_participants (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			event_id     UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			telegram_id  BIGINT NOT NULL REFERENCES users(telegram_id),
			status       TEXT NOT NULL DEFAULT 'pending',
			requested_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
			responded_at TIMESTAMPTZ,
			UNIQUE(event_id, telegram_id)
		);
		CREATE INDEX IF NOT EXISTS idx_ep_event ON event_participants(event_id);
		CREATE INDEX IF NOT EXISTS idx_ep_user ON event_participants(telegram_id);

		CREATE TABLE IF NOT EXISTS event_views (
			telegram_id BIGINT NOT NULL REFERENCES users(telegram_id),
			event_id    UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			viewed_at   TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
			PRIMARY KEY (telegram_id, event_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("migration 002 up: %w", err)
	}
	return nil
}

func down002(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		DROP TABLE IF EXISTS event_views;
		DROP TABLE IF EXISTS event_participants;
		DROP TABLE IF EXISTS events;
	`)
	if err != nil {
		return fmt.Errorf("migration 002 down: %w", err)
	}
	return nil
}
