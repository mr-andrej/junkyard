#!/bin/bash
# Build script for JUNKyard - cross-compiling on Windows to Linux

set -e

echo "🗑️  Building JUNKyard..."
echo ""

# Check if Docker is available for cross-compilation
if command -v docker &> /dev/null; then
    echo "🐳 Using Docker for cross-compilation..."
    
    docker run --rm -v "$(pwd)":/workspace -w /workspace golang:1.21 bash -c "
        apt-get update && apt-get install -y gcc-x86-64-linux-gnu > /dev/null 2>&1
        export CC=x86_64-linux-gnu-gcc
        export CGO_ENABLED=1
        export GOOS=linux
        export GOARCH=amd64
        
        mkdir -p bin
        echo '🔨 Building junkyard-server...'
        go build -ldflags='-s -w -X main.Version=1.0.0' -o bin/junkyard-server ./cmd/junkyard-server
        
        echo '🔨 Building junkyard CLI (junk)...'
        go build -ldflags='-s -w -X main.Version=1.0.0' -o bin/junk ./cmd/junkyard-cli
        
        chmod +x bin/*
    "
else
    echo "⚠️  Docker not found. Using local Go toolchain..."
    echo "   (This will work on Linux/macOS but not for cross-compilation)"
    
    mkdir -p bin
    echo '🔨 Building junkyard-server...'
    go build -ldflags='-s -w -X main.Version=1.0.0' -o bin/junkyard-server ./cmd/junkyard-server
    
    echo '🔨 Building junkyard CLI (junk)...'
    go build -ldflags='-s -w -X main.Version=1.0.0' -o bin/junk ./cmd/junkyard-cli
    
    chmod +x bin/* || true
fi

echo ""
echo "✅ Build complete!"
echo "   📦 bin/junkyard-server (server daemon)"
echo "   📦 bin/junk (CLI tool)"
echo ""
echo "Ready to deploy! 🚀"
