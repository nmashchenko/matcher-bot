package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GeoResult struct {
	Country     string
	CountryCode string
	State       string
	City        string
	IsUSA       bool
}

type nominatimResponse struct {
	Address struct {
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
		State       string `json:"state"`
		City        string `json:"city"`
		Town        string `json:"town"`
		Village     string `json:"village"`
	} `json:"address"`
}

// Geocoder performs reverse-geocoding via Nominatim.
type Geocoder struct {
	client  *http.Client
	baseURL string
}

// NewGeocoder creates a Geocoder with sensible defaults.
func NewGeocoder() *Geocoder {
	return &Geocoder{
		client:  &http.Client{Timeout: 10 * time.Second},
		baseURL: "https://nominatim.openstreetmap.org",
	}
}

// NewGeocoderWithURL creates a Geocoder pointing at a custom URL (useful for tests).
func NewGeocoderWithURL(baseURL string, client *http.Client) *Geocoder {
	return &Geocoder{client: client, baseURL: baseURL}
}

// ReverseGeocode resolves lat/lon to a country, state, and city.
func (g *Geocoder) ReverseGeocode(ctx context.Context, lat, lon float64) (*GeoResult, error) {
	url := fmt.Sprintf("%s/reverse?format=json&lat=%f&lon=%f&addressdetails=1", g.baseURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "MatcherBot/1.0")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocoding HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding: unexpected status %d", resp.StatusCode)
	}

	var nr nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&nr); err != nil {
		return nil, fmt.Errorf("geocoding: decode response: %w", err)
	}

	city := nr.Address.City
	if city == "" {
		city = nr.Address.Town
	}
	if city == "" {
		city = nr.Address.Village
	}

	return &GeoResult{
		Country:     nr.Address.Country,
		CountryCode: nr.Address.CountryCode,
		State:       nr.Address.State,
		City:        city,
		IsUSA:       nr.Address.CountryCode == "us",
	}, nil
}
