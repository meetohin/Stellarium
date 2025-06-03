.PHONY: build run test clean docker-up docker-down proto migrate

# Build all services
build:
	@echo "Building all services..."
	@for service in cmd/*; do \
		if [ -d "$$service" ]; then \
			echo "Building $$service..."; \
			go build -o bin/$$(basename $$service) ./$$service; \
		fi \
	done

# Run specific service
run-auth:
	go run ./cmd/auth-service

run-user:
	go run ./cmd/user-service

run-gateway:
	go run ./cmd/api-gateway

# Development environment
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Database operations
migrate-up:
	go run ./scripts/db/migrate.go up

migrate-down:
	go run ./scripts/db/migrate.go down

migrate-create:
	@read -p "Enter migration name: " name; \
	go run ./scripts/db/migrate.go create $$name

# Protocol Buffers
# Protocol Buffers
proto:
	@echo "Generating protobuf files..."
	@chmod +x scripts/generate_proto.sh
	@./scripts/generate_proto.sh

proto-install:
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto-clean:
	@echo "Cleaning generated protobuf files..."
	rm -f api/proto/auth/*.pb.go


# Testing
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Code quality
lint:
	golangci-lint run

fmt:
	go fmt ./...

# Clean
clean:
	rm -rf bin/
	rm -rf dist/
	go clean -cache

# Install development tools
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
