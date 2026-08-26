include .env
export

.PHONY: run build test test-cover lint tidy docker-up docker-down \
        migrate-up migrate-down migrate-force migrate-version migrate-create

run:
# 	cd api && MIGRATIONS_PATH=$(CURDIR)/db/migrations go run ./cmd/server
	cd api && MIGRATIONS_PATH=$(CURDIR)/../db/migrations go run ./cmd

build:
	cd api && go build -o bin/api ./cmd

test:
	cd api && go test ./...

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint:
	go vet ./...

tidy:
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