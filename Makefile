.PHONY: build build-api build-worker run-api run-worker test clean docker-up docker-down migrate-up migrate-down

# Build commands
build: build-api build-worker

build-api:
	go build -o bin/api cmd/api/main.go

build-worker:
	go build -o bin/worker cmd/worker/main.go

# Run commands
run-api: build-api
	./bin/api

run-worker: build-worker
	./bin/worker

# Test commands
test:
	go test -v ./...

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Migration commands
migrate-up:
	migrate -path migrations -database "postgres://notification_user:notification_password@localhost:5432/notification_db?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://notification_user:notification_password@localhost:5432/notification_db?sslmode=disable" down

# Development setup
dev-setup: docker-up
	sleep 10
	make migrate-up

# Clean commands
clean:
	rm -rf bin/
	docker-compose down -v

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build both API and worker"
	@echo "  build-api     - Build API server"
	@echo "  build-worker  - Build worker service"
	@echo "  run-api       - Build and run API server"
	@echo "  run-worker    - Build and run worker service"
	@echo "  test          - Run tests"
	@echo "  docker-up     - Start Docker services"
	@echo "  docker-down   - Stop Docker services"
	@echo "  migrate-up    - Run database migrations"
	@echo "  migrate-down  - Rollback database migrations"
	@echo "  dev-setup     - Setup development environment"
	@echo "  clean         - Clean build artifacts and Docker volumes"