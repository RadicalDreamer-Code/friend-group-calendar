.DEFAULT_GOAL := help
COMPOSE := docker compose

# Pick up port overrides from .env for the URLs printed by `make up`.
-include .env
export

.PHONY: help up down logs ps psql test typecheck gen-api reset-db

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

up: ## Build and start the whole stack (web, api, db, mailpit)
	$(COMPOSE) up --build --renew-anon-volumes -d
	@echo ""
	@echo "  Web app   http://localhost:$${WEB_PORT:-8082}"
	@echo "  API       http://localhost:$${API_PORT:-8080}/healthz"
	@echo "  Mailpit   http://localhost:$${MAILPIT_UI_PORT:-8025}"

down: ## Stop the stack (data is kept)
	$(COMPOSE) down

logs: ## Follow logs of all services
	$(COMPOSE) logs -f

ps: ## Show running services
	$(COMPOSE) ps

psql: ## Open a SQL shell on the local database
	$(COMPOSE) exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

test: ## Run Go tests and vet inside the api container
	$(COMPOSE) run --rm --no-deps api sh -c 'go vet ./... && go test ./...'

typecheck: ## Type-check the client inside the web container
	$(COMPOSE) run --rm --no-deps web npx tsc --noEmit

gen-api: ## Regenerate the typed API client from packages/api/openapi.yaml
	$(COMPOSE) run --rm --no-deps web npm run gen:api

reset-db: ## Delete the local database volume (all local data is lost)
	$(COMPOSE) down -v
