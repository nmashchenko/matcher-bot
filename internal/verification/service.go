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
	Country  string
	Error    string // "geocoding_failed" or ""
}

type Service struct {
	users     database.UserRepository
	geo       *geocoding.Geocoder
	whitelist database.WhitelistRepository
}

func NewService(users database.UserRepository, geo *geocoding.Geocoder, whitelist database.WhitelistRepository) *Service {
	return &Service{users: users, geo: geo, whitelist: whitelist}
}

func (s *Service) VerifyByLocation(ctx context.Context, telegramID int64, lat, lon float64) (*VerifyResult, error) {
	whitelisted, err := s.whitelist.IsWhitelisted(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	if whitelisted {
		now := time.Now()
		defaultLat := geocoding.DefaultLat
		defaultLon := geocoding.DefaultLon
		country := geocoding.DefaultCountry
		state := geocoding.DefaultState
		city := geocoding.DefaultCity
		userState := database.StateOnboarding
		err = s.users.Update(ctx, telegramID, &database.UserUpdateData{
			UserState:  &userState,
			Latitude:   &defaultLat,
			Longitude:  &defaultLon,
			Country:    &country,
			State:      &state,
			City:       &city,
			VerifiedAt: &now,
		})
		if err != nil {
			return nil, err
		}
		return &VerifyResult{
			Verified: true,
			City:     city,
			State:    state,
		}, nil
	}

	loc, err := s.geo.ReverseGeocode(ctx, lat, lon)
	if err != nil {
		return &VerifyResult{Verified: false, Error: "geocoding_failed"}, nil
	}

	if loc.IsUSA {
		now := time.Now()
		state := database.StateOnboarding
		err = s.users.Update(ctx, telegramID, &database.UserUpdateData{
			UserState: &state,
			Latitude:  &lat,
			Longitude: &lon,
			Country:   &loc.Country,
			State:     &loc.State,
			City:      &loc.City,
			VerifiedAt: &now,
		})
		if err != nil {
			return nil, err
		}
		return &VerifyResult{
			Verified: true,
			State:    loc.State,
			City:     loc.City,
		}, nil
	}

	// Not in USA — user stays unverified, can retry
	return &VerifyResult{Verified: false, Country: loc.Country}, nil
}
