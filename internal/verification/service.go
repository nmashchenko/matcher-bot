package verification

import (
	"context"
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/geocoding"
)

type VerifyResult struct {
	Verified bool
	City     string
	State    string
	Error    string // "geocoding_failed" or ""
}

type Service struct {
	users database.UserRepository
	geo   *geocoding.Geocoder
}

func NewService(users database.UserRepository, geo *geocoding.Geocoder) *Service {
	return &Service{users: users, geo: geo}
}

func (s *Service) VerifyByLocation(ctx context.Context, telegramID int64, lat, lon float64) (*VerifyResult, error) {
	geo, err := s.geo.ReverseGeocode(ctx, lat, lon)
	if err != nil {
		return &VerifyResult{Verified: false, Error: "geocoding_failed"}, nil
	}

	if geo.IsUSA {
		now := time.Now()
		state := database.StateOnboarding
		err = s.users.Update(ctx, telegramID, &database.UserUpdateData{
			UserState: &state,
			Latitude:  &lat,
			Longitude: &lon,
			Country:   &geo.Country,
			State:     &geo.State,
			City:      &geo.City,
			VerifiedAt: &now,
		})
		if err != nil {
			return nil, err
		}
		return &VerifyResult{
			Verified: true,
			State:    geo.State,
			City:     geo.City,
		}, nil
	}

	// Not in USA — user stays unverified, can retry
	return &VerifyResult{Verified: false}, nil
}
