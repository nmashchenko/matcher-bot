.PHONY: help run build migrate db-reset seed

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## Start the bot locally
	go run ./cmd/bot

build: ## Build the bot binary
	go build -o matcher-bot ./cmd/bot

migrate: ## Run database migrations (up)
	go run ./migrations up

db-reset: ## Drop all tables and re-run migrations
	go run ./migrations reset

seed: ## Seed test user + sample events
	go run ./scripts/seed.go
