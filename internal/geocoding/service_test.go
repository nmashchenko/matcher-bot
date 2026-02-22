package geocoding

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReverseGeocode_USA(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	// Override the URL by replacing the function temporarily
	origFunc := reverseGeocodeURL
	reverseGeocodeURL = func(lat, lon float64) string { return server.URL }
	defer func() { reverseGeocodeURL = origFunc }()

	result, err := ReverseGeocode(37.7749, -122.4194)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"address": {
				"country": "Germany",
				"country_code": "de",
				"state": "Berlin",
				"city": "Berlin"
			}
		}`))
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	origFunc := reverseGeocodeURL
	reverseGeocodeURL = func(lat, lon float64) string { return server.URL }
	defer func() { reverseGeocodeURL = origFunc }()

	result, err := ReverseGeocode(52.52, 13.405)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.IsUSA {
		t.Error("expected IsUSA = false")
	}
}

func TestReverseGeocode_CityFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"address": {
				"country": "United States",
				"country_code": "us",
				"state": "Vermont",
				"town": "Stowe"
			}
		}`))
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	origFunc := reverseGeocodeURL
	reverseGeocodeURL = func(lat, lon float64) string { return server.URL }
	defer func() { reverseGeocodeURL = origFunc }()

	result, err := ReverseGeocode(44.46, -72.68)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.City != "Stowe" {
		t.Errorf("City = %q; want Stowe (town fallback)", result.City)
	}
}

func TestReverseGeocode_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	origFunc := reverseGeocodeURL
	reverseGeocodeURL = func(lat, lon float64) string { return server.URL }
	defer func() { reverseGeocodeURL = origFunc }()

	result, err := ReverseGeocode(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result on server error, got %+v", result)
	}
}
