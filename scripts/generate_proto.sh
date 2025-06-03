#!/bin/bash

# scripts/generate_proto.sh - Script to generate protobuf files

set -e

echo "🔧 Generating protobuf files..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc not found. Please install Protocol Buffers compiler."
    echo "On macOS: brew install protobuf"
    echo "On Ubuntu: sudo apt-get install protobuf-compiler"
    exit 1
fi

# Check if Go protoc plugins are installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "❌ protoc-gen-go not found. Installing..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "❌ protoc-gen-go-grpc not found. Installing..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Add Go bin to PATH if not already there
export PATH="$PATH:$(go env GOPATH)/bin"

# Create output directory if it doesn't exist
mkdir -p api/proto/auth

# Generate Go files from proto
echo "Generating Go files from auth.proto..."
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/proto/auth/auth.proto

# Check if files were generated successfully
if [[ -f "api/proto/auth/auth.pb.go" && -f "api/proto/auth/auth_grpc.pb.go" ]]; then
    echo "✅ Protobuf files generated successfully:"
    echo "   - api/proto/auth/auth.pb.go"
    echo "   - api/proto/auth/auth_grpc.pb.go"
else
    echo "❌ Failed to generate protobuf files"
    exit 1
fi

echo "🎉 Proto generation completed!"