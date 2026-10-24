.DEFAULT_GOAL := help
COMPOSE := docker compose

# Pick up port overrides from .env for the URLs printed by `make up`.
-include .env
export

.PHONY: help up down logs ps psql test typecheck lint gen-api gen-sql reset-db

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

up: ## Build and start the whole stack (web, api, db, mailpit, s3)
	$(COMPOSE) up --build --renew-anon-volumes -d
	@echo ""
	@echo "  Web app   http://localhost:$${WEB_PORT:-8082}"
	@echo "  API       http://localhost:$${API_PORT:-8080}/healthz"
	@echo "  Mailpit   http://localhost:$${MAILPIT_UI_PORT:-8025}"
	@echo "  S3        http://localhost:$${S3_CONSOLE_PORT:-9001}/rustfs/console/"

down: ## Stop the stack (data is kept)
	$(COMPOSE) down

logs: ## Follow logs of all services
	$(COMPOSE) logs -f

ps: ## Show running services
	$(COMPOSE) ps

psql: ## Open a SQL shell on the local database
	$(COMPOSE) exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

test: ## Run Go vet and tests (including database tests) inside the api container
	$(COMPOSE) up -d --wait db
	$(COMPOSE) run --rm --no-deps api sh -c 'go vet ./... && go test ./...'

typecheck: ## Type-check the client inside the web container
	$(COMPOSE) run --rm --no-deps web npx tsc --noEmit

lint: ## ESLint, Prettier and the theme contrast check for the client
	$(COMPOSE) run --rm --no-deps web sh -c 'npm run lint && npx prettier --check src scripts && npm run check:contrast'

gen-api: ## Regenerate the typed API client from packages/api/openapi.yaml
	$(COMPOSE) run --rm --no-deps web npm run gen:api

gen-sql: ## Regenerate the Go database code from the SQL in services/api/internal/database
	docker run --rm -v "$(CURDIR)/services/api:/src" -w /src sqlc/sqlc:1.30.0 generate

reset-db: ## Delete the local database volume (all local data is lost)
	$(COMPOSE) down -v
