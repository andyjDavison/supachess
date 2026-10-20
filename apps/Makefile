ifneq (,$(wildcard .env))
	include .env
	export
endif

.PHONY: run-server run-frontend build-server build-frontend test test-server test-frontend test-watch-frontend lint lint-server lint-frontend tidy-server docker-up docker-down migrate-up migrate-down migrate-force migrate-version migrate-create

run-server:
# 	cd api && MIGRATIONS_PATH=$(CURDIR)/db/migrations go run ./cmd/server
	cd api && go run ./cmd

run-frontend:
	cd ui && pnpm run dev

build-server:
	cd api && go build -o bin/api ./cmd

build-frontend:
	cd ui && pnpm run build

# Both suites, the way CI should run them. -race matters on the server:
# Hub, Queue and TimerManager are all touched from several goroutines.
test: test-server test-frontend

test-server:
	cd api && go test ./... -race

test-frontend:
	cd ui && pnpm run test:run

# `pnpm test` is bare `vitest`, which watches and never exits - fine at a
# terminal, a hang in CI. That's why test-frontend uses test:run instead.
test-watch-frontend:
	cd ui && pnpm run test

lint: lint-server lint-frontend

# Note the `cd api`: there is no go.mod at apps/, so `go vet ./...` from
# here fails with "directory prefix . does not contain main module".
lint-server:
	cd api && go vet ./...

lint-frontend:
	cd ui && pnpm run lint

tidy-server:
	go mod tidy

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down

# Use if a migration failed partway through and left the schema_migrations
# table in a "dirty" state — forces it back to a clean version number
# without re-running the SQL. version=<N> required, e.g. make migrate-force version=1
migrate-force:
	migrate -path db/migrations -database "$(DATABASE_URL)" force $(version)

migrate-version:
	migrate -path db/migrations -database "$(DATABASE_URL)" version

# Scaffolds a new numbered up/down pair. name=<description> required,
# e.g. make migrate-create name=add_users_bio_column
migrate-create:
	migrate create -ext sql -dir db/migrations -seq $(name)