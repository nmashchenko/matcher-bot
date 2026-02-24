package verification

import (
	"context"
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/geocoding"
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

type geocoder interface {
	ReverseGeocode(ctx context.Context, lat, lon float64) (*geocoding.GeoResult, error)
}

type Service struct {
	users database.UserRepository
	geo   geocoder
}

func NewService(users database.UserRepository, geo geocoder) *Service {
	return &Service{users: users, geo: geo}
}

func (s *Service) VerifyByLocation(ctx context.Context, telegramID int64, lat, lon float64) (*VerifyResult, error) {
	geo, err := s.geo.ReverseGeocode(ctx, lat, lon)
	if err != nil {
		return &VerifyResult{Verified: false, Error: "geocoding_failed"}, nil
	}

	now := time.Now()

	if geo.IsUSA {
		status := database.StatusVerified
		err = s.users.Update(ctx, telegramID, &database.UserUpdateData{
			VerificationStatus: &status,
			Latitude:           &lat,
			Longitude:          &lon,
			Country:            &geo.Country,
			State:              &geo.State,
			City:               &geo.City,
			VerifiedAt:         &now,
		})
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
	status := database.StatusRejected
	err = s.users.Update(ctx, telegramID, &database.UserUpdateData{
		VerificationStatus: &status,
	})
	if err != nil {
		return nil, err
	}

	return &VerifyResult{Verified: false, Status: database.StatusRejected}, nil
}

func (s *Service) GetVerificationStatus(ctx context.Context, telegramID int64) (*StatusResult, error) {
	user, err := s.users.GetByTelegramID(ctx, telegramID)
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
