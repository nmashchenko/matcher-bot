# Matcher Bot

Telegram bot that matches CIS youth living in the US — location-verified onboarding, AI-powered profile embeddings, and 48-hour expiring match chats.

## Tech Stack

- **Go 1.24** + [telebot v4](https://github.com/go-telegram/telebot)
- **PostgreSQL 17** + [pgvector](https://github.com/pgvector/pgvector) (HNSW cosine indices)
- **OpenAI** `text-embedding-3-small` (1536-dim embeddings for bio & looking-for)
- **Nominatim** (OpenStreetMap) for reverse geocoding
- **Docker** multi-stage build + Docker Compose

## Quick Start

```bash
cp .env.example .env   # fill in DATABASE_URL, TELEGRAM_BOT_TOKEN, OPENAI_API_KEY
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

## Implemented

- [x] **Geolocation verification** — one-time location share, Nominatim reverse geocode, US-only gate
- [x] **User state machine** — `unverified` → `onboarding` → `ready`
- [x] **5-step onboarding** — age (auto-detect from Telegram birthday), goal (inline buttons), bio, looking-for, profile summary
- [x] **Context-aware prompts** — dynamic messages that reference the user's city, age, goal, and name as they progress
- [x] **Welcome sticker** — sends a sticker at onboarding start (configurable file_id)
- [x] **AI embeddings** — bio and looking-for texts embedded via OpenAI, stored as pgvector columns with HNSW indices
- [x] **Avatar extraction** — grabs profile photo from Telegram automatically
- [x] **Resume onboarding** — returns user to the exact incomplete step on reconnect
- [x] **Sticker debug handler** — send any sticker to get its `file_id` back
- [x] **Database migrations** — versioned up/down/reset/status system
- [x] **Docker deployment** — multi-stage Dockerfile + Compose with Postgres 17

## TODO

- [ ] **Matching engine** — browse profiles one-by-one, cosine similarity ranking using stored embeddings
- [ ] **Text reactions** — free-text like/pass/maybe/report with LLM-based intent + tag extraction
- [ ] **Preference learning** — tag weights from reactions, rejection reason tracking, feed re-ranking
- [ ] **"Who liked you" feed** — transparent incoming likes (no paywall), instant match on mutual like
- [ ] **48-hour match chats** — auto-created 2-person rooms with bot ice-breaker, auto-close after 48h
- [ ] **Session dynamics** — streak detection, playful comments on rejection runs, "yesterday memory" summaries
- [ ] **Rating & badges** — like rate, match rate, response rate, selectivity balance, behavioral badges (Ghost, Attention Hunter, etc.)
- [ ] **Profile viewing** — `/profile` command to view/edit your own card
- [ ] **Re-verification** — periodic location re-check for long-inactive users
- [ ] **Monetization** — daily view/like limits, Telegram Stars payments, profile boost, rewind/second-chance

## Design Docs

Detailed specs for planned features live in [`docs/`](docs/):

- [Matching System](docs/matching.md) — card browsing, text reactions, mutual likes, "who liked you"
- [48-Hour Match Chats](docs/match-chats.md) — auto-created rooms, bot ice-breaker script, auto-close
- [Personalization](docs/personalization.md) — preference learning, session dynamics, "yesterday memory"
- [Rating & Badges](docs/rating-and-badges.md) — quality signals, behavioral badges, how they affect visibility
- [Monetization](docs/monetization.md) — free limits, paid expansions via Telegram Stars
