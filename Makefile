.PHONY: build release test test-root coverage coverage-root lint clean run deps help

# Version can be overridden: make build VERSION=v1.0.0
VERSION ?= dev

# ldflags for the version stamp (used by all build targets)
VERSION_LDFLAGS := -X main.Version=$(VERSION)

# ldflags for release builds: strip DWARF + symbol table for smaller binaries
RELEASE_LDFLAGS := -s -w $(VERSION_LDFLAGS)

# Build the application (dev build: keeps debug info, allows CGO defaults)
# Use `make release` for stripped, statically-linked production binaries.
build:
	go build -ldflags="$(VERSION_LDFLAGS)" -o udp-sender .

# Build a release binary (CGO disabled, debug info stripped)
# Matches what release.yml and the Dockerfile produce.
release:
	CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o udp-sender .

# Run tests (without root - some tests will skip)
test:
	go test -v ./...

# Run tests with root privileges
test-root:
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "This target requires root privileges. Run: sudo make test-root"; \
		exit 1; \
	fi
	go test -v -count=1 ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with coverage as root (bypasses test cache)
coverage-root:
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "This target requires root privileges. Run: sudo make coverage-root"; \
		exit 1; \
	fi
	go test -v -race -count=1 -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter (installs golangci-lint v1 if not present)
lint:
	@GOLANGCI_LINT=$$(command -v golangci-lint 2>/dev/null || echo "$$(go env GOPATH)/bin/golangci-lint"); \
	if [ ! -x "$$GOLANGCI_LINT" ]; then \
		echo "golangci-lint not found. Installing v1.62.2..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.62.2; \
		GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	fi; \
	$$GOLANGCI_LINT run

# Clean build artifacts
clean:
	rm -f udp-sender coverage.out coverage.html

# Run the application (requires root)
run: build
	@echo "Note: This requires root privileges"
	@echo "Run: sudo make run"
	./udp-sender

# Install dependencies
deps:
	go mod download
	go mod verify

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build a dev binary (keeps debug info, allows CGO)"
	@echo "                   Set VERSION to override version string"
	@echo "                   Example: make build VERSION=v1.0.0"
	@echo "  release        - Build a release binary (CGO_ENABLED=0, stripped)"
	@echo "                   Matches release workflow / Dockerfile output"
	@echo "                   Example: make release VERSION=v1.0.0"
	@echo "  test           - Run tests (without root, some will skip)"
	@echo "  test-root      - Run all tests with root privileges (bypasses cache)"
	@echo "  coverage       - Run tests with coverage report"
	@echo "  coverage-root  - Run tests with coverage as root (includes raw socket tests)"
	@echo "  lint           - Run golangci-lint (auto-installs if not present)"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run the application (requires root)"
	@echo "  deps           - Download and verify dependencies"
	@echo "  help           - Show this help message"

