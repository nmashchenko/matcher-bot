.PHONY: help db-reset db-migrate db-studio dev

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

db-reset: ## Drop all tables and re-run migrations
	npx prisma migrate reset --force

db-migrate: ## Run pending migrations
	npx prisma migrate dev

db-studio: ## Open Prisma Studio
	npx prisma studio

dev: ## Start the bot in dev mode
	npx nest start --watch
