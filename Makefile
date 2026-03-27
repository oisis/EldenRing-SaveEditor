# Variables
BINARY_NAME=ER-Save-Editor
VERSION=0.1.0
BUILD_DIR=build/bin

.PHONY: all build dev test lint clean extract-data deps help

all: deps build test

# Install dependencies for both Go and Frontend
deps:
	@echo "📥 Installing dependencies..."
	go mod download
	cd frontend && npm install

# Build the application for the current platform
build:
	@echo "🔨 Building $(BINARY_NAME) v$(VERSION)..."
	wails build -o $(BINARY_NAME)

# Run Wails in development mode (hot reload)
dev:
	wails dev

# Run all tests
test:
	@echo "🧪 Running unit tests..."
	go test -v ./backend/...
	@echo "🧪 Running round-trip validation tests..."
	go test -v ./tests/roundtrip_test.go

# Run linter (requires golangci-lint installed)
lint:
	@echo "🔍 Running linter..."
	golangci-lint run ./...

# Tool to extract data from Rust source to Go (Phase 2)
extract-data:
	@echo "📂 Extracting game data from Rust source..."
	go run scripts/extractor.go tmp/org-src/src/db/ backend/db/data/

# Clean build artifacts
clean:
	@echo "🧹 Cleaning up..."
	rm -rf build/bin/*
	rm -rf frontend/dist

# Help command
help:
	@echo "Available commands:"
	@echo "  make deps         - Install Go and Frontend dependencies"
	@echo "  make build        - Build the app for current platform"
	@echo "  make dev          - Run app in development mode"
	@echo "  make test         - Run all tests"
	@echo "  make lint         - Run linter"
	@echo "  make extract-data - Extract DB from Rust to Go"
	@echo "  make clean        - Remove build artifacts"
