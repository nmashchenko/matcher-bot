package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	pgvector "github.com/pgvector/pgvector-go"
	"github.com/uptrace/bun"
)

// UpdateData holds optional user fields to update. Nil means "don't touch".
type UserUpdateData struct {
	VerificationStatus  *VerificationStatus
	Latitude            *float64
	Longitude           *float64
	Country             *string
	State               *string
	City                *string
	VerifiedAt          *time.Time
	AvatarFileID        *string
	Age                 *int
	Goal                *Goal
	Bio                 *string
	LookingFor          *string
	BioEmbedding        *pgvector.Vector
	LookingForEmbedding *pgvector.Vector
	OnboardingStep      *OnboardingStep
}

// UserRepository is the interface satisfied by UserStore.
// Consumers should depend on this rather than on *UserStore directly.
type UserRepository interface {
	FindOrCreate(ctx context.Context, telegramID int64, username, firstName, lastName *string) (*User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
	Update(ctx context.Context, telegramID int64, data *UserUpdateData) error
}

type UserStore struct {
	db *bun.DB
}

func NewUserStore(db *bun.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) FindOrCreate(ctx context.Context, telegramID int64, username, firstName, lastName *string) (*User, error) {
	user := new(User)
	err := s.db.NewSelect().
		Model(user).
		Where("telegram_id = ?", telegramID).
		Scan(ctx)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("find user: %w", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		user = &User{
			TelegramID: telegramID,
			Username:   username,
			FirstName:  firstName,
			LastName:   lastName,
		}
		_, err = s.db.NewInsert().Model(user).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
		return user, nil
	}

	// Update existing user info
	user.Username = username
	user.FirstName = firstName
	user.LastName = lastName
	user.UpdatedAt = time.Now()
	_, err = s.db.NewUpdate().
		Model(user).
		Column("username", "first_name", "last_name", "updated_at").
		Where("telegram_id = ?", telegramID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

func (s *UserStore) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	user := new(User)
	err := s.db.NewSelect().
		Model(user).
		Where("telegram_id = ?", telegramID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserStore) Update(ctx context.Context, telegramID int64, data *UserUpdateData) error {
	q := s.db.NewUpdate().
		TableExpr("users").
		Where("telegram_id = ?", telegramID).
		Set("updated_at = ?", time.Now())

	if data.VerificationStatus != nil {
		q = q.Set("verification_status = ?", string(*data.VerificationStatus))
	}
	if data.Latitude != nil {
		q = q.Set("latitude = ?", *data.Latitude)
	}
	if data.Longitude != nil {
		q = q.Set("longitude = ?", *data.Longitude)
	}
	if data.Country != nil {
		q = q.Set("country = ?", *data.Country)
	}
	if data.State != nil {
		q = q.Set("state = ?", *data.State)
	}
	if data.City != nil {
		q = q.Set("city = ?", *data.City)
	}
	if data.VerifiedAt != nil {
		q = q.Set("verified_at = ?", *data.VerifiedAt)
	}
	if data.AvatarFileID != nil {
		q = q.Set("avatar_file_id = ?", *data.AvatarFileID)
	}
	if data.Age != nil {
		q = q.Set("age = ?", *data.Age)
	}
	if data.Goal != nil {
		q = q.Set("goal = ?", string(*data.Goal))
	}
	if data.Bio != nil {
		q = q.Set("bio = ?", *data.Bio)
	}
	if data.LookingFor != nil {
		q = q.Set("looking_for = ?", *data.LookingFor)
	}
	if data.BioEmbedding != nil {
		q = q.Set("bio_embedding = ?", *data.BioEmbedding)
	}
	if data.LookingForEmbedding != nil {
		q = q.Set("looking_for_embedding = ?", *data.LookingForEmbedding)
	}
	if data.OnboardingStep != nil {
		q = q.Set("onboarding_step = ?", string(*data.OnboardingStep))
	}

	_, err := q.Exec(ctx)
	return err
}
