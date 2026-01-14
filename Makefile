.PHONY: all build clean test run proto help

# Build variables
BINARY_DIR=bin
MAIN_PACKAGES=cmd/node cmd/wallet cmd/node-p2p cmd/node-grpc cmd/web-server

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build all binaries
all: clean build

# Build all executables
build:
	@echo "Building all binaries..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/node cmd/node/main.go
	$(GOBUILD) -o $(BINARY_DIR)/wallet cmd/wallet/main.go
	$(GOBUILD) -o $(BINARY_DIR)/node-p2p cmd/node-p2p/main.go
	$(GOBUILD) -o $(BINARY_DIR)/node-grpc cmd/node-grpc/main.go
	$(GOBUILD) -o $(BINARY_DIR)/web-server cmd/web-server/main.go
	@echo "✅ Build complete! Binaries in $(BINARY_DIR)/"

# Build individual binaries
node:
	$(GOBUILD) -o $(BINARY_DIR)/node cmd/node/main.go

wallet:
	$(GOBUILD) -o $(BINARY_DIR)/wallet cmd/wallet/main.go

node-p2p:
	$(GOBUILD) -o $(BINARY_DIR)/node-p2p cmd/node-p2p/main.go
node-grpc:
	$(GOBUILD) -o $(BINARY_DIR)/node-grpc cmd/node-grpc/main.go

web-server:
	$(GOBUILD) -o $(BINARY_DIR)/web-server cmd/web-server/main.go
	$(GOBUILD) -o $(BINARY_DIR)/node-grpc cmd/node-grpc/main.go

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -short ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover ./...

test-all:
	@echo "Running all tests (including P2P)..."
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -rf data/
	rm -f *.db
	rm -rf node*.db
	@echo "✅ Clean complete!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies installed!"

# Regenerate protobuf files
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/blockchain.proto
	@echo "✅ Protobuf files generated!"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "✅ Code formatted!"

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...
	@echo "✅ Code vetted!"

# Run basic node
run:
	$(BINARY_DIR)/node

# Run gRPC node
# Run P2P node
run-p2p:
	$(BINARY_DIR)/node-p2p -listen "/ip4/0.0.0.0/tcp/9001" -fresh -mine

# Run web server (requires gRPC node running)
run-web:
	$(BINARY_DIR)/web-server
# Run P2P node
run-p2p:
	$(BINARY_DIR)/node-p2p -listen "/ip4/0.0.0.0/tcp/9001" -fresh -mine

# Help
help:
	@echo "Bitcoin-Like Blockchain - Makefile Commands"
	@echo ""
	@echo "Building:"
	@echo "  make build       - Build all binaries"
	@echo "  make node        - Build node binary"
	@echo "  make wallet      - Build wallet binary"
	@echo "  make node-p2p    - Build P2P node binary"
	@echo "  make node-grpc   - Build gRPC node binary"
	@echo "  make web-server  - Build web server binary"
	@echo ""
	@echo "Testing:"
	@echo "  make test        - Run tests (short mode)"
	@echo "  make test-all    - Run all tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo ""
	@echo "Running:"
	@echo "  make run         - Run basic node"
	@echo "  make run-grpc    - Run gRPC node"
	@echo "  make run-p2p     - Run P2P node"
	@echo "  make run-web     - Run web server (requires gRPC node)"
	@echo ""
	@echo "Development:"
	@echo "  make deps        - Install dependencies"
	@echo "  make proto       - Regenerate protobuf files"
	@echo "  make fmt         - Format code"
	@echo "  make vet         - Vet code"
	@echo "  make clean       - Clean build artifacts"
	@echo ""
	@echo "Other:"
	@echo "  make help        - Show this help message"
