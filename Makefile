.PHONY: build run test clean

# Postgres Env var defaults (see docker-compose.yml)
PGUSER ?= postgres
PGPASS ?= postgres
PGHOST ?= 127.0.0.1
PGPORT ?= 5432
PGDB ?= simple-web-app-test

DB_URL ?= "postgres://$(PGUSER):$(PGPASS)@$(PGHOST)/$(PGDB)?sslmode=disable"

build:
	go build -ldflags "-s -w" -o simple-web-app cmd/simple-web-app/main.go

run: build
	./simple-web-app

test:
	go test -v ./...

clean:
	rm -rf simple-web-app

sqlc-generate:
	cd internal && sqlc generate

db-migrate: #: Run database migrations
	@echo "Creating DB if missing (ignoring errors)"
	- PGPASSWORD=$(PGPASS) createdb \
			-U $(PGUSER) \
			-h $(PGHOST) \
			$(PGDB)
	@echo "Migrating..."
	@docker run --rm --network host -v "$(CURDIR)/internal/db/migrations/":/db/migrations migrate/migrate --path=/db/migrations -database $(DB_URL) up 1

db-rollback: #: Rollback database migrations
	@docker run --rm --network host -v "$(CURDIR)/internal/db/migrations/":/db/migrations migrate/migrate --path=/db/migrations -database $(DB_URL) down 1

db-drop: #: Drop db
	@docker run --rm --network host -v "$(CURDIR)/internal/db/migrations/":/db/migrations migrate/migrate --path=/db/migrations -database $(DB_URL) drop -f
