#!/bin/bash

# TradingBotHub - Project Setup Script
# Создает базовую структуру микросервисного проекта

echo "🚀 Setting up TradingBotHub project structure..."

# Create main project directory
mkdir -p trading-bot-platform
cd trading-bot-platform

# Create directory structure
mkdir -p cmd/{api-gateway,auth-service,user-service,bot-service,market-data-service,execution-service,risk-service,analytics-service,notification-service}
mkdir -p internal/{auth,database,grpc,config,middleware,utils}
mkdir -p pkg/{trading,indicators,exchanges,risk,events}
mkdir -p api/{proto,openapi}
mkdir -p deployments/{docker,kubernetes,helm}
mkdir -p scripts/{build,deploy,db}
mkdir -p docs/{api,architecture,deployment}
mkdir -p web/{frontend,admin}
mkdir -p configs/{local,dev,prod}

# Initialize Go modules
go mod init github.com/tradingbothub/platform

# Create basic .gitignore
cat > .gitignore << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
cmd/*/main
dist/
build/

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment files
.env
.env.local
.env.*.local

# IDE
.vscode/
.idea/
*.swp
*.swo

# Logs
*.log
logs/

# Database
*.db
*.sqlite

# Docker
.dockerignore
Dockerfile.local

# OS
.DS_Store
Thumbs.db

# Secrets
secrets/
certs/
keys/
EOF

# Create main go.mod with dependencies
cat > go.mod << 'EOF'
module github.com/tradingbothub/platform

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.5.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0
	github.com/influxdata/influxdb-client-go/v2 v2.13.0
	github.com/lib/pq v1.10.9
	github.com/nats-io/nats.go v1.31.0
	github.com/redis/go-redis/v9 v9.3.1
	github.com/shopspring/decimal v1.3.1
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.17.0
	golang.org/x/crypto v0.17.0
	google.golang.org/grpc v1.60.1
	google.golang.org/protobuf v1.31.0
	gorm.io/driver/postgres v1.5.4
	gorm.io/gorm v1.25.5
)
EOF

# Create docker-compose for local development
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: tradingbot
      POSTGRES_USER: tradingbot
      POSTGRES_PASSWORD: tradingbot123
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/db/init.sql:/docker-entrypoint-initdb.d/init.sql

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  influxdb:
    image: influxdb:2.7-alpine
    environment:
      DOCKER_INFLUXDB_INIT_MODE: setup
      DOCKER_INFLUXDB_INIT_USERNAME: admin
      DOCKER_INFLUXDB_INIT_PASSWORD: tradingbot123
      DOCKER_INFLUXDB_INIT_ORG: tradingbothub
      DOCKER_INFLUXDB_INIT_BUCKET: market_data
    ports:
      - "8086:8086"
    volumes:
      - influxdb_data:/var/lib/influxdb2

  nats:
    image: nats:2.10-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - nats_data:/data

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./deployments/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

volumes:
  postgres_data:
  redis_data:
  influxdb_data:
  nats_data:
  prometheus_data:
EOF

# Create Makefile for development
cat > Makefile << 'EOF'
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
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go-grpc_out=. api/proto/**/*.proto

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
EOF

echo "✅ Basic project structure created!"
echo "📁 Project structure:"
tree -L 3 -a

echo ""
echo "🔧 Next steps:"
echo "1. cd trading-bot-platform"
echo "2. make install-tools"
echo "3. make docker-up"
echo "4. Start developing services!"
