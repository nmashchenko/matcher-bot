package verification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"matcher-bot/internal/database"
	"matcher-bot/internal/geocoding"
)

type mockWhitelistStore struct {
	whitelisted bool
}

func (m *mockWhitelistStore) IsWhitelisted(_ context.Context, _ int64) (bool, error) {
	return m.whitelisted, nil
}

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

func newTestGeo(t *testing.T, handler http.HandlerFunc) *geocoding.Geocoder {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return geocoding.NewGeocoderWithURL(server.URL, server.Client())
}

func TestVerifyByLocation_USA(t *testing.T) {
	geo := newTestGeo(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"address": {
				"country": "United States",
				"country_code": "us",
				"state": "California",
				"city": "San Francisco"
			}
		}`))
	})
	svc := NewService(&mockUserStore{}, geo, &mockWhitelistStore{})

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
	geo := newTestGeo(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"address": {
				"country": "Germany",
				"country_code": "de",
				"state": "Berlin",
				"city": "Berlin"
			}
		}`))
	})
	svc := NewService(&mockUserStore{}, geo, &mockWhitelistStore{})

	result, err := svc.VerifyByLocation(context.Background(), 12345, 52.52, 13.40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verified {
		t.Error("expected Verified = false")
	}
}

func TestVerifyByLocation_Whitelisted(t *testing.T) {
	// Geocoder returns non-USA, but user is whitelisted — should still verify.
	geo := newTestGeo(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("geocoder should not be called for whitelisted users")
	})
	svc := NewService(&mockUserStore{}, geo, &mockWhitelistStore{whitelisted: true})

	result, err := svc.VerifyByLocation(context.Background(), 454332651, 55.75, 37.61)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verified {
		t.Error("expected Verified = true for whitelisted user")
	}
	if result.State != "Hawaii" {
		t.Errorf("State = %q; want Hawaii", result.State)
	}
	if result.City != "Kahaluu-Keauhou" {
		t.Errorf("City = %q; want Kahaluu-Keauhou", result.City)
	}
}

func TestVerifyByLocation_GeocodingFailed(t *testing.T) {
	geo := newTestGeo(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	svc := NewService(&mockUserStore{}, geo, &mockWhitelistStore{})

	result, err := svc.VerifyByLocation(context.Background(), 12345, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != "geocoding_failed" {
		t.Errorf("Error = %q; want geocoding_failed", result.Error)
	}
}
