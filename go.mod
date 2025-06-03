module github.com/tradingbothub/platform

go 1.23

toolchain go1.23.5

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0
	github.com/influxdata/influxdb-client-go/v2 v2.13.0
	github.com/lib/pq v1.10.9
	github.com/nats-io/nats.go v1.31.0
	github.com/redis/go-redis/v9 v9.3.1
	github.com/shopspring/decimal v1.3.1
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.17.0
	golang.org/x/crypto v0.33.0
	google.golang.org/grpc v1.72.2
	google.golang.org/protobuf v1.36.6
	gorm.io/driver/postgres v1.5.4
	gorm.io/gorm v1.25.5
)

require golang.org/x/sys v0.30.0 // indirect
