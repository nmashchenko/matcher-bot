package verification

import (
	"context"
	"fmt"
	"testing"

	"matcher-bot/internal/database"
	"matcher-bot/internal/geocoding"
)

type mockUserStore struct {
	user *database.User
	err  error
}

func (m *mockUserStore) FindOrCreate(_ context.Context, _ int64, _, _, _ *string) (*database.User, error) {
	return m.user, m.err
}

func (m *mockUserStore) GetByTelegramID(_ context.Context, _ int64) (*database.User, error) {
	return m.user, m.err
}

func (m *mockUserStore) Update(_ context.Context, _ int64, _ *database.UserUpdateData) error {
	return m.err
}

type mockGeocoder struct {
	result *geocoding.GeoResult
	err    error
}

func (m *mockGeocoder) ReverseGeocode(_ context.Context, _, _ float64) (*geocoding.GeoResult, error) {
	return m.result, m.err
}

func TestVerifyByLocation_USA(t *testing.T) {
	svc := NewService(
		&mockUserStore{},
		&mockGeocoder{result: &geocoding.GeoResult{
			Country: "United States", CountryCode: "us",
			State: "California", City: "San Francisco", IsUSA: true,
		}},
	)

	result, err := svc.VerifyByLocation(context.Background(), 12345, 37.77, -122.41)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verified {
		t.Error("expected Verified = true")
	}
	if result.State != "California" {
		t.Errorf("State = %q; want California", result.State)
	}
}

func TestVerifyByLocation_NonUSA(t *testing.T) {
	svc := NewService(
		&mockUserStore{},
		&mockGeocoder{result: &geocoding.GeoResult{
			Country: "Germany", CountryCode: "de",
			State: "Berlin", City: "Berlin", IsUSA: false,
		}},
	)

	result, err := svc.VerifyByLocation(context.Background(), 12345, 52.52, 13.40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verified {
		t.Error("expected Verified = false")
	}
	if result.Status != database.StatusRejected {
		t.Errorf("Status = %q; want REJECTED", result.Status)
	}
}

func TestVerifyByLocation_GeocodingFailed(t *testing.T) {
	svc := NewService(
		&mockUserStore{},
		&mockGeocoder{err: fmt.Errorf("network error")},
	)

	result, err := svc.VerifyByLocation(context.Background(), 12345, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != "geocoding_failed" {
		t.Errorf("Error = %q; want geocoding_failed", result.Error)
	}
}

func TestGetVerificationStatus(t *testing.T) {
	state := "California"
	city := "SF"
	svc := NewService(
		&mockUserStore{user: &database.User{
			VerificationStatus: database.StatusVerified,
			State:              &state,
			City:               &city,
		}},
		nil,
	)

	status, err := svc.GetVerificationStatus(context.Background(), 12345)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != database.StatusVerified {
		t.Errorf("Status = %q; want VERIFIED", status.Status)
	}
}
