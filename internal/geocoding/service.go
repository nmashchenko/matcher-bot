package geocoding

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

var httpClient = &http.Client{}

var reverseGeocodeURL = func(lat, lon float64) string {
	return fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f&addressdetails=1",
		lat, lon,
	)
}

func ReverseGeocode(lat, lon float64) (*GeoResult, error) {
	url := reverseGeocodeURL(lat, lon)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "MatcherBot/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("geocoding HTTP error: %v", err)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("geocoding: non-200 status %d", resp.StatusCode)
		return nil, nil
	}

	var nr nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&nr); err != nil {
		log.Printf("geocoding: decode error: %v", err)
		return nil, nil
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
