# Matcher Bot

Telegram bot for organizing local events among CIS youth living in the US — location-verified onboarding, event creation, browse & join with host approval.

## Tech Stack

- **Go 1.24** + [telebot v4](https://github.com/go-telegram/telebot)
- **PostgreSQL 17** via [bun ORM](https://github.com/uptrace/bun)
- **Nominatim** (OpenStreetMap) for reverse geocoding
- **Docker** multi-stage build + Docker Compose

## Quick Start

```bash
cp .env.example .env   # fill in DATABASE_URL, TELEGRAM_BOT_TOKEN
make migrate            # run database migrations
make run                # start the bot
```

## Make Commands

| Command | Description |
|---------|-------------|
| `make run` | Start the bot locally |
| `make build` | Build the binary |
| `make migrate` | Run database migrations |
| `make db-reset` | Drop all tables and re-migrate |

## How It Works

1. **Verification** — user shares location once, Nominatim confirms they're in the US (whitelisted users bypass geo-check)
2. **Onboarding** — age collected (auto-detected from Telegram birthday when available), user state transitions `unverified` → `onboarding` → `ready`
3. **Create events** — 6-step wizard: type, title, description, date/time, location, capacity + optional age restrictions (min/max)
4. **Browse events** — swipe through active events in your city (filterable by type), gaming events visible across all cities, age-gated events hidden if you're outside the range
5. **Join & approve** — send a join request, host gets notified and approves/rejects
6. **Manage events** — view participants, remove people, cancel events; participants get real-time notifications
7. **Settings** — choose a preferred event type filter to only see events you care about

### Privacy & Location

- **Your precise location is not stored.** It is used only once during verification to confirm you are in the United States. Coordinates are reverse-geocoded on the fly to determine your city and state, and are not persisted.
- **Only your city and state are saved** — solely to match you with nearby events. No precise GPS data is retained.
- **Age** is collected during onboarding solely for age-gated event filtering. It is not displayed to other users.
- **No data is sold or shared externally.** All data stays within the bot's database and is used only to power event matching.

## Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Begin verification / resume onboarding / main menu |
| `/events` | Browse events nearby |
| `/create` | Create a new event |
| `/myevents` | View hosted & joined events |
| `/settings` | Set preferred event type filter |

## Project Structure

```
cmd/bot/            — entrypoint
internal/
  bot/              — bot setup, middleware, top-level handlers
  database/         — models, repositories (users, events, participants)
  events/           — event creation wizard, browsing, management
  geocoding/        — Nominatim reverse geocoding client
  messages/         — all user-facing strings (Russian)
  onboarding/       — age collection flow
  settings/         — user preference management
  ptr/              — pointer helpers (Str, Deref)
  verification/     — location-based US verification
migrations/         — versioned database migrations
```

## Event Types

| Type | Emoji | City-scoped |
|------|-------|-------------|
| Hangout (Тусовка) | 🤙 | Yes |
| Gaming (Игры) | 🎮 | No (all cities) |
| Sports (Спорт) | ⚽ | Yes |
| Concert (Концерт / Шоу) | 🎵 | Yes |
| Random (Рандом) | 🎲 | Yes |

## Testing

```bash
go test ./... -v
```

Unit tests cover pointer helpers, event time parsing, callback data parsing, event type lookups, and all message formatting functions. Database tests require `DATABASE_URL` and are skipped otherwise.
