.PHONY: help build clean test install docker-build lint fmt vet run-server run-cli

# Variables
BINARY_SERVER=bin/junkyard-server
BINARY_CLI=bin/junk
VERSION=1.0.0
GO=go
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

help:
	@echo "🗑️  JUNKyard Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build         - Build both server and CLI (current platform)"
	@echo "  make build-linux   - Cross-compile to Linux (requires Docker)"
	@echo "  make build-server  - Build server only"
	@echo "  make build-cli     - Build CLI only"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make run-server    - Run server locally (requires SQLite)"
	@echo "  make run-cli       - Run CLI client"
	@echo "  make install       - Install binaries to GOPATH/bin"
	@echo "  make docker-build  - Build Docker image"

build: build-server build-cli

build-server:
	@echo "🔨 Building junkyard-server..."
	@mkdir -p bin
	$(GO) build $(LDFLAGS) -o $(BINARY_SERVER) ./cmd/junkyard-server

build-cli:
	@echo "🔨 Building junk CLI..."
	@mkdir -p bin
	$(GO) build $(LDFLAGS) -o $(BINARY_CLI) ./cmd/junkyard-cli

build-linux:
	@echo "🐳 Cross-compiling for Linux (Docker)..."
	@docker run --rm -v "$$(pwd)":/workspace -w /workspace golang:1.21 bash -c " \
		apt-get update && apt-get install -y gcc-x86-64-linux-gnu && \
		export CC=x86_64-linux-gnu-gcc && \
		export CGO_ENABLED=1 && \
		export GOOS=linux && \
		export GOARCH=amd64 && \
		mkdir -p bin && \
		echo '🔨 Building junkyard-server...' && \
		go build $(LDFLAGS) -o $(BINARY_SERVER) ./cmd/junkyard-server && \
		echo '🔨 Building junk CLI...' && \
		go build $(LDFLAGS) -o $(BINARY_CLI) ./cmd/junkyard-cli && \
		chmod +x bin/* \
	"
	@echo "✅ Build complete!"

clean:
	@echo "🧹 Cleaning up..."
	@rm -rf bin/
	@$(GO) clean
	@rm -f junkyard.db* junkyard-wal

test:
	@echo "🧪 Running tests..."
	$(GO) test -v -cover ./...

lint:
	@echo "🔍 Running linter..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run ./...

fmt:
	@echo "🎨 Formatting code..."
	$(GO) fmt ./...

vet:
	@echo "🔎 Running go vet..."
	$(GO) vet ./...

run-server: build-server
	@echo "🚀 Starting JUNKyard server..."
	@./$(BINARY_SERVER) --http-addr :8080 --syslog-addr :5514 --db-path ./junkyard.db --retention-days 14

run-cli: build-cli
	@./$(BINARY_CLI) logs

install: build
	@echo "📦 Installing binaries..."
	$(GO) install ./cmd/junkyard-server
	$(GO) install ./cmd/junkyard-cli
	@echo "✅ Installed to: $$($(GO) env GOPATH)/bin"

docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t junkyard:$(VERSION) .

.PHONY: deps
deps:
	@echo "📚 Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

.PHONY: version
version:
	@echo "JUNKyard v$(VERSION)"
