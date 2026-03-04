SHELL := /bin/bash

DB_USER=postgres
DB_PASS=postgres
DB_NAME=marketplace
DB_HOST=localhost
DB_PORT=55432
DB_URL=postgresql://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: up down ps logs migrate-up db-shell generate


up:
	docker compose -f deploy/docker-compose.yml up -d --build

down:
	docker compose -f deploy/docker-compose.yml down

db-shell:
	docker compose -f deploy/docker-compose.yml exec -it db psql -U $(DB_USER) -d $(DB_NAME)

migrate-up:
	docker compose -f deploy/docker-compose.yml run --rm flyway migrate

ps:
	docker compose -f deploy/docker-compose.yml ps

logs:
	docker compose -f deploy/docker-compose.yml logs -f --tail=200

generate:
	oapi-codegen --config oapi-codegen.yaml api/openapi.yaml