package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"matcher-bot/internal/geocoding"

	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	seedTelegramID = 424873509
	seedUsername    = "nmashchenko"
	seedFirstName  = "Nikita"
	seedLastName   = "Mashchenko"
	seedCity    = geocoding.DefaultCity
	seedState   = geocoding.DefaultState
	seedCountry = geocoding.DefaultCountry
	seedLat     = geocoding.DefaultLat
	seedLon     = geocoding.DefaultLon
	seedAge        = 24
	seedAvatarID   = "AgACAgIAAxUAAWmsgVkQH0m3zr3l1cbjWDIF7tmaAAIBC2sbJQ5TGTKbxpyO1ULiAQADAgADYwADOgQ"
)

type seedEvent struct {
	Title        string
	Description  string
	EventType    string
	City         string
	State        string
	Lat          float64
	Lon          float64
	Capacity     int
	HoursFromNow int
	MinAge       *int
	MaxAge       *int
}

func intPtr(v int) *int { return &v }

var seedEvents = []seedEvent{
	{
		Title:        "Рандомная встреча",
		Description:  "Познакомимся и посмотрим, что из этого выйдет",
		EventType:    "random",
		City:         seedCity, State: seedState, Lat: seedLat, Lon: seedLon,
		Capacity:     15,
		HoursFromNow: 48,
		MinAge:       intPtr(18),
		MaxAge:       intPtr(30),
	},
	{
		Title:        "Волейбол на пляже",
		Description:  "Дружеская игра, уровень — любой",
		EventType:    "sports",
		City:         seedCity, State: seedState, Lat: seedLat, Lon: seedLon,
		Capacity:     8,
		HoursFromNow: 24,
	},
	{
		Title:        "CS2 вечер",
		Description:  "Competitive 5v5, нужен микрофон",
		EventType:    "gaming",
		City:         seedCity, State: seedState, Lat: seedLat, Lon: seedLon,
		Capacity:     10,
		HoursFromNow: 6,
	},
	{
		Title:        "Кофе и знакомство",
		Description:  "",
		EventType:    "hangout",
		City:         seedCity, State: seedState, Lat: seedLat, Lon: seedLon,
		Capacity:     5,
		HoursFromNow: 72,
		MinAge:       intPtr(21),
		MaxAge:       intPtr(35),
	},
	{
		Title:        "Концерт в баре",
		Description:  "Живая музыка, вход свободный",
		EventType:    "concert",
		City:         seedCity, State: seedState, Lat: seedLat, Lon: seedLon,
		Capacity:     20,
		HoursFromNow: 96,
	},
}

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	ctx := context.Background()

	// Upsert seed user.
	now := time.Now()
	_, err := db.NewRaw(`
		INSERT INTO users (telegram_id, username, first_name, last_name, user_state,
			latitude, longitude, country, state, city, avatar_file_id, age,
			verified_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'ready', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			user_state = EXCLUDED.user_state,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			country = EXCLUDED.country,
			state = EXCLUDED.state,
			city = EXCLUDED.city,
			avatar_file_id = EXCLUDED.avatar_file_id,
			age = EXCLUDED.age,
			updated_at = EXCLUDED.updated_at
	`,
		seedTelegramID, seedUsername, seedFirstName, seedLastName,
		seedLat, seedLon, seedCountry, seedState, seedCity,
		seedAvatarID, seedAge,
		now, now, now,
	).Exec(ctx)
	if err != nil {
		log.Fatalf("upsert user: %v", err)
	}
	fmt.Printf("User @%s (telegram_id=%d) ready.\n", seedUsername, seedTelegramID)

	// Insert events.
	for _, ev := range seedEvents {
		var desc interface{} = nil
		if ev.Description != "" {
			desc = ev.Description
		}
		startsAt := now.Add(time.Duration(ev.HoursFromNow) * time.Hour)

		_, err := db.NewRaw(`
			INSERT INTO events (host_telegram_id, title, description, event_type, event_state,
				latitude, longitude, city, state, max_participants, min_age, max_age, starts_at,
				created_at, updated_at)
			VALUES (?, ?, ?, ?, 'active', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			seedTelegramID, ev.Title, desc, ev.EventType,
			ev.Lat, ev.Lon, ev.City, ev.State,
			ev.Capacity, ev.MinAge, ev.MaxAge, startsAt,
			now, now,
		).Exec(ctx)
		if err != nil {
			log.Fatalf("insert event %q: %v", ev.Title, err)
		}
		fmt.Printf("  Created: %s (%s) — starts in %dh\n", ev.Title, ev.EventType, ev.HoursFromNow)
	}

	fmt.Printf("\nDone: 1 user + %d events seeded.\n", len(seedEvents))
}
