package database

import (
	"context"
	"time"

	pgvector "github.com/pgvector/pgvector-go"
	"github.com/uptrace/bun"
)

// UpdateData holds optional user fields to update. Nil means "don't touch".
type UserUpdateData struct {
	AvatarFileID        *string
	Age                 *int
	Goal                *Goal
	Bio                 *string
	LookingFor          *string
	BioEmbedding        *pgvector.Vector
	LookingForEmbedding *pgvector.Vector
	OnboardingStep      *OnboardingStep
}

type UserStore struct {
	DB *bun.DB
}

func NewUserStore(db *bun.DB) *UserStore {
	return &UserStore{DB: db}
}

func (s *UserStore) FindOrCreate(ctx context.Context, telegramID int64, username, firstName, lastName *string) (*User, error) {
	user := new(User)
	err := s.DB.NewSelect().
		Model(user).
		Where("telegram_id = ?", telegramID).
		Scan(ctx)

	if err != nil {
		// User not found, create new
		user = &User{
			TelegramID: telegramID,
			Username:   username,
			FirstName:  firstName,
			LastName:   lastName,
		}
		_, err = s.DB.NewInsert().Model(user).Exec(ctx)
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	// Update existing user info
	user.Username = username
	user.FirstName = firstName
	user.LastName = lastName
	user.UpdatedAt = time.Now()
	_, err = s.DB.NewUpdate().
		Model(user).
		Column("username", "first_name", "last_name", "updated_at").
		Where("telegram_id = ?", telegramID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserStore) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	user := new(User)
	err := s.DB.NewSelect().
		Model(user).
		Where("telegram_id = ?", telegramID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserStore) Update(ctx context.Context, telegramID int64, data *UserUpdateData) error {
	q := s.DB.NewUpdate().
		TableExpr("users").
		Where("telegram_id = ?", telegramID).
		Set("updated_at = ?", time.Now())

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
