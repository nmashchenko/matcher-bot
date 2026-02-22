package verification

import (
	"context"
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/geocoding"

	"github.com/uptrace/bun"
)

type VerifyResult struct {
	Verified bool
	Status   database.VerificationStatus
	State    string
	City     string
	Error    string // "geocoding_failed" or ""
}

type StatusResult struct {
	Status database.VerificationStatus
	State  string
	City   string
}

type Service struct {
	DB *bun.DB
}

func NewService(db *bun.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) FindOrCreateUser(ctx context.Context, telegramID int64, username, firstName, lastName *string) (*database.User, error) {
	user := new(database.User)
	err := s.DB.NewSelect().
		Model(user).
		Where("telegram_id = ?", telegramID).
		Scan(ctx)

	if err != nil {
		// User not found, create new
		user = &database.User{
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

func (s *Service) VerifyByLocation(ctx context.Context, telegramID int64, lat, lon float64) (*VerifyResult, error) {
	geo, err := geocoding.ReverseGeocode(lat, lon)
	if err != nil || geo == nil {
		return &VerifyResult{Verified: false, Error: "geocoding_failed"}, nil
	}

	now := time.Now()

	if geo.IsUSA {
		_, err = s.DB.NewUpdate().
			TableExpr("users").
			Set("verification_status = ?", database.StatusVerified).
			Set("latitude = ?", lat).
			Set("longitude = ?", lon).
			Set("country = ?", geo.Country).
			Set("state = ?", geo.State).
			Set("city = ?", geo.City).
			Set("verified_at = ?", now).
			Set("updated_at = ?", now).
			Where("telegram_id = ?", telegramID).
			Exec(ctx)
		if err != nil {
			return nil, err
		}
		return &VerifyResult{
			Verified: true,
			Status:   database.StatusVerified,
			State:    geo.State,
			City:     geo.City,
		}, nil
	}

	// Not in USA
	_, err = s.DB.NewUpdate().
		TableExpr("users").
		Set("verification_status = ?", database.StatusRejected).
		Set("updated_at = ?", now).
		Where("telegram_id = ?", telegramID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &VerifyResult{Verified: false, Status: database.StatusRejected}, nil
}

func (s *Service) GetVerificationStatus(ctx context.Context, telegramID int64) (*StatusResult, error) {
	user := new(database.User)
	err := s.DB.NewSelect().
		Model(user).
		Column("verification_status", "state", "city").
		Where("telegram_id = ?", telegramID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := &StatusResult{
		Status: user.VerificationStatus,
	}
	if user.State != nil {
		result.State = *user.State
	}
	if user.City != nil {
		result.City = *user.City
	}
	return result, nil
}
