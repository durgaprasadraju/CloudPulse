# Copyright 2026 Durga Prasad Raju Nadimpalli
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: dev dev-build down logs ps clean backend-run frontend-install frontend-dev help

COMPOSE := docker compose -f deploy/docker-compose.yml
BACKEND_DIR := backend
FRONTEND_DIR := frontend

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

dev: ## Start all services (postgres, backend, frontend)
	$(COMPOSE) up

dev-build: ## Rebuild images and start all services
	$(COMPOSE) up --build

down: ## Stop and remove containers
	$(COMPOSE) down

logs: ## Tail logs from all services
	$(COMPOSE) logs -f

ps: ## List running compose services
	$(COMPOSE) ps

clean: ## Stop containers and remove volumes
	$(COMPOSE) down -v

backend-run: ## Run Go API locally on :8080
	cd $(BACKEND_DIR) && go run ./cmd/beacon

backend-test: ## Run backend tests
	cd $(BACKEND_DIR) && go test ./...

frontend-install: ## Install frontend dependencies
	cd $(FRONTEND_DIR) && npm install

frontend-dev: ## Run Vite dev server on :5173
	cd $(FRONTEND_DIR) && npm run dev

frontend-build: ## Build frontend for production
	cd $(FRONTEND_DIR) && npm run build
