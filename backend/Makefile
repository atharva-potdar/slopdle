.PHONY: all build run test migrate seed clean sqlc

all: build

sqlc:
	~/go/bin/sqlc generate

build: sqlc
	go build -o bin/server main.go

run: build
	./bin/server

migrate:
	mariadb -u atharva-potdar test < ../db/migrations/001_schema.up.sql

migrate-down:
	mariadb -u atharva-potdar test < ../db/migrations/001_schema.down.sql

seed:
	mariadb -u atharva-potdar test < ../db/migrations/002_seed.up.sql

seed-down:
	mariadb -u atharva-potdar test < ../db/migrations/002_seed.down.sql

reset-db: migrate-down migrate seed
	@echo "Database reset and seeded."

clean:
	rm -rf bin/
