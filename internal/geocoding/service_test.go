package geocoding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestGeocoder(t *testing.T, handler http.HandlerFunc) *Geocoder {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Geocoder{client: server.Client(), baseURL: server.URL}
}

func TestReverseGeocode_USA(t *testing.T) {
	t.Parallel()
	g := newTestGeocoder(t, func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != "MatcherBot/1.0" {
			t.Errorf("User-Agent = %q; want MatcherBot/1.0", ua)
		}
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

	result, err := g.ReverseGeocode(context.Background(), 37.7749, -122.4194)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsUSA {
		t.Error("expected IsUSA = true")
	}
	if result.State != "California" {
		t.Errorf("State = %q; want California", result.State)
	}
	if result.City != "San Francisco" {
		t.Errorf("City = %q; want San Francisco", result.City)
	}
}

func TestReverseGeocode_NonUSA(t *testing.T) {
	t.Parallel()
	g := newTestGeocoder(t, func(w http.ResponseWriter, r *http.Request) {
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

	result, err := g.ReverseGeocode(context.Background(), 52.52, 13.405)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsUSA {
		t.Error("expected IsUSA = false")
	}
}

func TestReverseGeocode_CityFallback(t *testing.T) {
	t.Parallel()
	g := newTestGeocoder(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"address": {
				"country": "United States",
				"country_code": "us",
				"state": "Vermont",
				"town": "Stowe"
			}
		}`))
	})

	result, err := g.ReverseGeocode(context.Background(), 44.46, -72.68)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.City != "Stowe" {
		t.Errorf("City = %q; want Stowe (town fallback)", result.City)
	}
}

func TestReverseGeocode_ServerError(t *testing.T) {
	t.Parallel()
	g := newTestGeocoder(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := g.ReverseGeocode(context.Background(), 0, 0)
	if err == nil {
		t.Fatal("expected error on server error, got nil")
	}
}
