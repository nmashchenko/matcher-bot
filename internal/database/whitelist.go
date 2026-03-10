package database

import (
	"context"

	"github.com/uptrace/bun"
)

type WhitelistRepository interface {
	IsWhitelisted(ctx context.Context, telegramID int64) (bool, error)
}

type WhitelistStore struct {
	db *bun.DB
}

func NewWhitelistStore(db *bun.DB) *WhitelistStore {
	return &WhitelistStore{db: db}
}

func (s *WhitelistStore) IsWhitelisted(ctx context.Context, telegramID int64) (bool, error) {
	exists, err := s.db.NewSelect().
		TableExpr("whitelist").
		Where("telegram_id = ?", telegramID).
		Exists(ctx)
	return exists, err
}
