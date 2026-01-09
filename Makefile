.PHONY: build test clean run-controller run-worker migrate-up migrate-down docker-up docker-down build-runtime-images

# Build all services
build:
	@echo "Building services..."
	go build -o bin/controller cmd/controller/main.go
	go build -o bin/worker cmd/worker/main.go

# Build runtime images
build-runtime-images:
	@echo "Building runtime images..."
	cd runtime-images && chmod +x build.sh && ./build.sh

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf storage/

# Run controller
run-controller:
	@echo "Starting controller..."
	go run cmd/controller/main.go

# Run worker
run-worker:
	@echo "Starting worker..."
	go run cmd/worker/main.go

# Database migrations
migrate-up:
	@echo "Running migrations..."
	psql $(DB_DSN) -f migrations/001_initial_schema.up.sql

migrate-down:
	@echo "Rolling back migrations..."
	psql $(DB_DSN) -f migrations/001_initial_schema.down.sql

# Docker compose commands
docker-up:
	@echo "Starting infrastructure..."
	docker-compose up -d postgres redis

docker-down:
	@echo "Stopping infrastructure..."
	docker-compose down

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Run all checks
check: fmt lint test

# Development setup
dev-setup: docker-up
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Running migrations..."
	@make migrate-up
	@echo "Building runtime images..."
	@make build-runtime-images
	@echo "Development environment ready!"
