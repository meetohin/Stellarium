// internal/auth/service_test.go
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// Mock repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Mock token service
type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateAccessToken(userID string) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) GenerateRefreshToken(userID string) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateAccessToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateRefreshToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func TestService_Register(t *testing.T) {
	mockRepo := new(MockRepository)
	mockTokenService := new(MockTokenService)
	service := NewService(mockRepo, mockTokenService)

	ctx := context.Background()
	req := &RegisterRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	// Mock user not exists
	mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, ErrUserNotFound)
	
	// Mock successful user creation
	mockRepo.On("Create", ctx, mock.AnythingOfType("*auth.User")).Return(nil)
	
	// Mock token generation
	mockTokenService.On("GenerateAccessToken", mock.AnythingOfType("string")).Return("access_token", nil)
	mockTokenService.On("GenerateRefreshToken", mock.AnythingOfType("string")).Return("refresh_token", nil)

	resp, err := service.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "access_token", resp.AccessToken)
	assert.Equal(t, "refresh_token", resp.RefreshToken)
	assert.Equal(t, req.Email, resp.User.Email)
	assert.Equal(t, req.Username, resp.User.Username)

	mockRepo.AssertExpectations(t)
	mockTokenService.AssertExpectations(t)
}

func TestService_Login_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockTokenService := new(MockTokenService)
	service := NewService(mockRepo, mockTokenService)

	ctx := context.Background()
	req := &LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	// Create test user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := &User{
		ID:           "user-123",
		Email:        req.Email,
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
	}

	// Mock user exists
	mockRepo.On("GetByEmail", ctx, req.Email).Return(user, nil)
	mockRepo.On("Update", ctx, user).Return(nil)
	
	// Mock token generation
	mockTokenService.On("GenerateAccessToken", user.ID).Return("access_token", nil)
	mockTokenService.On("GenerateRefreshToken", user.ID).Return("refresh_token", nil)

	resp, err := service.Login(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "access_token", resp.AccessToken)
	assert.Equal(t, "refresh_token", resp.RefreshToken)
	assert.Equal(t, user.Email, resp.User.Email)

	mockRepo.AssertExpectations(t)
	mockTokenService.AssertExpectations(t)
}

func TestService_Login_InvalidCredentials(t *testing.T) {
	mockRepo := new(MockRepository)
	mockTokenService := new(MockTokenService)
	service := NewService(mockRepo, mockTokenService)

	ctx := context.Background()
	req := &LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	// Create test user with different password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	user := &User{
		ID:           "user-123",
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	// Mock user exists
	mockRepo.On("GetByEmail", ctx, req.Email).Return(user, nil)

	resp, err := service.Login(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
	assert.Nil(t, resp)

	mockRepo.AssertExpectations(t)
}

// scripts/test/api_test.sh
#!/bin/bash

# API Testing Script for TradingBotHub
set -e

BASE_URL="http://localhost:8080/api/v1"

echo "🧪 Testing TradingBotHub API..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test API endpoint
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local auth_header=$5

    echo -e "${YELLOW}Testing: $method $endpoint${NC}"
    
    if [ -n "$auth_header" ]; then
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X $method \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $auth_header" \
            -d "$data" \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X $method \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint")
    fi

    body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    status=$(echo $response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')

    if [ "$status" -eq "$expected_status" ]; then
        echo -e "${GREEN}✅ PASS: Status $status${NC}"
        echo "Response: $body"
    else
        echo -e "${RED}❌ FAIL: Expected $expected_status, got $status${NC}"
        echo "Response: $body"
        exit 1
    fi
    echo ""
}

# Test health check
test_endpoint "GET" "/health" "" 200

# Test user registration
echo "🔧 Testing user registration..."
register_data='{
    "email": "test@example.com",
    "username": "testuser",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User"
}'

response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "$register_data" \
    "$BASE_URL/auth/register")

body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
status=$(echo $response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')

if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    echo -e "${GREEN}✅ Registration successful${NC}"
    echo "Response: $body"
else
    echo -e "${YELLOW}⚠️  Registration failed (user might already exist): Status $status${NC}"
fi

# Test user login
echo "🔑 Testing user login..."
login_data='{
    "email": "test@example.com",
    "password": "password123"
}'

response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "$login_data" \
    "$BASE_URL/auth/login")

body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
status=$(echo $response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')

if [ "$status" -eq 200 ]; then
    echo -e "${GREEN}✅ Login successful${NC}"
    # Extract access token
    access_token=$(echo $body | jq -r '.access_token')
    echo "Access Token: $access_token"
else
    echo -e "${RED}❌ Login failed: Status $status${NC}"
    echo "Response: $body"
    exit 1
fi

# Test protected endpoints with token
echo "🔒 Testing protected endpoints..."
test_endpoint "GET" "/user/profile" "" 200 "$access_token"
test_endpoint "GET" "/bots" "" 200 "$access_token"
test_endpoint "GET" "/strategies" "" 200 "$access_token"
test_endpoint "GET" "/portfolio" "" 200 "$access_token"

# Test token refresh
echo "🔄 Testing token refresh..."
refresh_data="{\"refresh_token\": \"$(echo $body | jq -r '.refresh_token')\"}"
test_endpoint "POST" "/auth/refresh" "$refresh_data" 200

# Test without token (should fail)
echo "🚫 Testing unauthorized access..."
response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X GET \
    -H "Content-Type: application/json" \
    "$BASE_URL/user/profile")

status=$(echo $response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')

if [ "$status" -eq 401 ]; then
    echo -e "${GREEN}✅ Unauthorized access properly blocked${NC}"
else
    echo -e "${RED}❌ Security issue: Unauthorized access allowed${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 All tests passed!${NC}"

# scripts/build/build.sh
#!/bin/bash

# Build script for TradingBotHub services
set -e

echo "🚀 Building TradingBotHub services..."

# Create build directory
mkdir -p bin

# Build all services
services=("auth-service" "api-gateway")

for service in "${services[@]}"; do
    echo "Building $service..."
    go build -o bin/$service ./cmd/$service
    echo "✅ $service built successfully"
done

# Build Docker images if Docker is available
if command -v docker &> /dev/null; then
    echo "🐳 Building Docker images..."
    
    for service in "${services[@]}"; do
        echo "Building Docker image for $service..."
        docker build -f deployments/docker/Dockerfile.$service -t tradingbothub/$service:latest .
        echo "✅ Docker image for $service built successfully"
    done
else
    echo "⚠️  Docker not found, skipping Docker image build"
fi

echo "🎉 Build completed successfully!"

# scripts/deploy/deploy.sh
#!/bin/bash

# Deployment script for TradingBotHub
set -e

ENVIRONMENT=${1:-local}

echo "🚀 Deploying TradingBotHub to $ENVIRONMENT environment..."

case $ENVIRONMENT in
    "local")
        echo "Starting local development environment..."
        docker-compose up -d
        echo "✅ Local environment started"
        ;;
    "dev")
        echo "Deploying to development environment..."
        kubectl apply -f deployments/kubernetes/namespace.yaml
        kubectl apply -f deployments/kubernetes/secrets.yaml
        kubectl apply -f deployments/kubernetes/
        echo "✅ Deployed to development"
        ;;
    "prod")
        echo "Deploying to production environment..."
        # Production deployment with additional checks
        read -p "Are you sure you want to deploy to production? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kubectl apply -f deployments/kubernetes/namespace.yaml
            kubectl apply -f deployments/kubernetes/secrets.yaml
            kubectl apply -f deployments/kubernetes/
            echo "✅ Deployed to production"
        else
            echo "❌ Production deployment cancelled"
            exit 1
        fi
        ;;
    *)
        echo "❌ Unknown environment: $ENVIRONMENT"
        echo "Usage: $0 [local|dev|prod]"
        exit 1
        ;;
esac

# Load testing script
# scripts/test/load_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 10 }, // Ramp up to 10 users
    { duration: '5m', target: 10 }, // Stay at 10 users
    { duration: '2m', target: 50 }, // Ramp up to 50 users
    { duration: '5m', target: 50 }, // Stay at 50 users
    { duration: '2m', target: 0 },  // Ramp down to 0 users
  ],
};

const BASE_URL = 'http://localhost:8080/api/v1';

export function setup() {
  // Register a test user
  const registerPayload = JSON.stringify({
    email: 'loadtest@example.com',
    username: 'loadtestuser',
    password: 'password123',
    first_name: 'Load',
    last_name: 'Test'
  });

  const registerParams = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  http.post(`${BASE_URL}/auth/register`, registerPayload, registerParams);

  // Login to get token
  const loginPayload = JSON.stringify({
    email: 'loadtest@example.com',
    password: 'password123'
  });

  const loginResponse = http.post(`${BASE_URL}/auth/login`, loginPayload, registerParams);
  const loginData = JSON.parse(loginResponse.body);
  
  return {
    accessToken: loginData.access_token,
  };
}

export default function(data) {
  const params = {
    headers: {
      'Authorization': `Bearer ${data.accessToken}`,
      'Content-Type': 'application/json',
    },
  };

  // Test different endpoints
  let response = http.get(`${BASE_URL}/user/profile`, params);
  check(response, {
    'profile status is 200': (r) => r.status === 200,
  });

  response = http.get(`${BASE_URL}/bots`, params);
  check(response, {
    'bots status is 200': (r) => r.status === 200,
  });

  response = http.get(`${BASE_URL}/strategies`, params);
  check(response, {
    'strategies status is 200': (r) => r.status === 200,
  });

  response = http.get(`${BASE_URL}/portfolio`, params);
  check(response, {
    'portfolio status is 200': (r) => r.status === 200,
  });

  sleep(1);
}

export function teardown(data) {
  // Cleanup can be done here if needed
  console.log('Load test completed');
}

# API Documentation - docs/api/openapi.yaml
openapi: 3.0.3
info:
  title: TradingBotHub API
  description: API for algorithmic trading platform
  version: 1.0.0
  contact:
    name: TradingBotHub Team
    email: api@tradingbothub.com
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT

servers:
  - url: http://localhost:8080/api/v1
    description: Development server
  - url: https://api.tradingbothub.com/v1
    description: Production server

paths:
  /health:
    get:
      summary: Health check
      operationId: healthCheck
      tags:
        - System
      responses:
        '200':
          description: Service is healthy
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: healthy
                  timestamp:
                    type: integer
                    example: 1640995200
                  service:
                    type: string
                    example: api-gateway

  /auth/register:
    post:
      summary: Register new user
      operationId: register
      tags:
        - Authentication
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RegisterRequest'
      responses:
        '201':
          description: User registered successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuthResponse'
        '400':
          description: Invalid request data
        '409':
          description: User already exists

  /auth/login:
    post:
      summary: Login user
      operationId: login
      tags:
        - Authentication
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LoginRequest'
      responses:
        '200':
          description: Login successful
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuthResponse'
        '401':
          description: Invalid credentials

  /auth/refresh:
    post:
      summary: Refresh access token
      operationId: refreshToken
      tags:
        - Authentication
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                refresh_token:
                  type: string
      responses:
        '200':
          description: Token refreshed successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuthResponse'
        '401':
          description: Invalid refresh token

  /user/profile:
    get:
      summary: Get user profile
      operationId: getUserProfile
      tags:
        - User
      security:
        - BearerAuth: []
      responses:
        '200':
          description: User profile retrieved
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '401':
          description: Unauthorized

  /bots:
    get:
      summary: List user's trading bots
      operationId: listBots
      tags:
        - Bots
      security:
        - BearerAuth: []
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            default: 20
            maximum: 100
        - name: offset
          in: query
          schema:
            type: integer
            default: 0
      responses:
        '200':
          description: List of bots
          content:
            application/json:
              schema:
                type: object
                properties:
                  bots:
                    type: array
                    items:
                      $ref: '#/components/schemas/Bot'
                  total:
                    type: integer
                  limit:
                    type: integer
                  offset:
                    type: integer

components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        username:
          type: string
        first_name:
          type: string
        last_name:
          type: string
        avatar:
          type: string
          format: uri
        is_active:
          type: boolean
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time

    RegisterRequest:
      type: object
      required:
        - email
        - username
        - password
        - first_name
        - last_name
      properties:
        email:
          type: string
          format: email
        username:
          type: string
          minLength: 3
          maxLength: 50
        password:
          type: string
          minLength: 8
        first_name:
          type: string
        last_name:
          type: string

    LoginRequest:
      type: object
      required:
        - email
        - password
      properties:
        email:
          type: string
          format: email
        password:
          type: string

    AuthResponse:
      type: object
      properties:
        access_token:
          type: string
        refresh_token:
          type: string
        user:
          $ref: '#/components/schemas/User'
        expires_in:
          type: integer

    Bot:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        description:
          type: string
        strategy_id:
          type: string
          format: uuid
        exchange:
          type: string
        status:
          type: string
          enum: [active, paused, stopped, error]
        config:
          type: object
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time

  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

# README.md template
# 🤖 TradingBotHub - Algorithmic Trading Platform

A high-performance, microservices-based platform for algorithmic trading with support for multiple exchanges, custom strategies, and advanced risk management.

## 🚀 Features

- **Microservices Architecture**: Scalable, maintainable service-oriented design
- **Multiple Exchange Support**: Binance, Coinbase, Kraken integration
- **Custom Strategy Development**: Python and Go strategy engines
- **Advanced Risk Management**: Real-time portfolio monitoring and protection
- **Real-time Market Data**: WebSocket streams and historical data
- **Backtesting Engine**: Strategy validation with historical data
- **Copy Trading**: Follow successful traders
- **Web & Mobile Apps**: Cross-platform user interfaces

## 🛠 Tech Stack

- **Backend**: Go 1.21+, gRPC, Protocol Buffers
- **Databases**: PostgreSQL, InfluxDB (time-series), Redis (cache)
- **Message Queue**: NATS JetStream
- **Container**: Docker, Kubernetes
- **Monitoring**: Prometheus, Grafana, Jaeger
- **Frontend**: React, Next.js, TypeScript

## 🏗 Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Development Setup

1. **Clone the repository**
```bash
git clone https://github.com/tradingbothub/platform.git
cd platform
```

2. **Install dependencies**
```bash
make install-tools
go mod download
```

3. **Start infrastructure**
```bash
make docker-up
```

4. **Run database migrations**
```bash
make migrate-up
```

5. **Start services**
```bash
# Terminal 1: Auth Service
make run-auth

# Terminal 2: API Gateway
make run-gateway
```

6. **Test the API**
```bash
chmod +x scripts/test/api_test.sh
./scripts/test/api_test.sh
```

### Using the API

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "trader@example.com",
    "username": "trader123",
    "password": "securepassword",
    "first_name": "John",
    "last_name": "Trader"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "trader@example.com",
    "password": "securepassword"
  }'

# Use the returned access_token for authenticated requests
curl -X GET http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## 📁 Project Structure

```
trading-bot-platform/
├── cmd/                    # Service entry points
│   ├── api-gateway/        # HTTP API Gateway
│   └── auth-service/       # Authentication service
├── internal/               # Private application code
│   ├── auth/              # Authentication logic
│   ├── config/            # Configuration management
│   ├── database/          # Database connections
│   ├── gateway/           # API Gateway logic
│   └── middleware/        # HTTP middleware
├── api/                   # API definitions
│   └── proto/             # Protocol Buffer definitions
├── pkg/                   # Public libraries
├── deployments/           # Docker & Kubernetes configs
├── scripts/               # Build and deployment scripts
└── docs/                  # Documentation
```

## 🧪 Testing

```bash
# Run unit tests
make test

# Run with coverage
make test-coverage

# Load testing
k6 run scripts/test/load_test.js

# Integration tests
./scripts/test/api_test.sh
```

## 🚢 Deployment

### Local Development
```bash
make docker-up
```

### Kubernetes
```bash
# Development
./scripts/deploy/deploy.sh dev

# Production
./scripts/deploy/deploy.sh prod
```

## 📊 Monitoring

- **Metrics**: http://localhost:9090 (Prometheus)
- **Logs**: `docker-compose logs -f`
- **Health**: http://localhost:8080/health

## 🔧 Configuration

Configuration files are located in `configs/`:
- `local.yaml` - Local development
- `dev.yaml` - Development environment  
- `prod.yaml` - Production environment

Environment variables can override config values:
```bash
export DB_PASSWORD=your-password
export JWT_SECRET=your-jwt-secret
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📧 Email: support@tradingbothub.com
- 💬 Discord: [TradingBotHub Community](https://discord.gg/tradingbothub)
- 📚 Documentation: [docs.tradingbothub.com](https://docs.tradingbothub.com)

---

⭐ **Star this repository if you find it useful!**