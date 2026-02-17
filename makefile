# Makefile for volumetric-backend

# Load environment variables from .env if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Database connection string (uses variables from .env)
DB_DSN := postgres://postgres:password@localhost:5432/volumetric_db?sslmode=disable

# Directory where migration files live
MIGRATIONS_DIR := internal/migrations

.PHONY: migrate-new migrate-up migrate-down migrate-force migrate-status 

# Create a new migration file
# Usage: make migrate-new name=add_some_field
migrate-new:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-new name=your_migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

# Apply all pending migrations
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

# Apply a specific number of migrations
# Usage: make migrate-up-n n=1
migrate-up-n:
	@if [ -z "$(n)" ]; then \
		echo "Usage: make migrate-up-n n=1"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up $(n)

# Roll back the last migration
migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1

# Force set the migration version (useful when version is out of sync)
# Usage: make migrate-force version=0
migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Usage: make migrate-force version=0"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" force $(version)

# Show current migration status/version
migrate-status:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" version
