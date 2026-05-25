.PHONY: proto build test lint dev clean

# Proto code generation
proto:
	buf generate

# Build all services
build:
	@echo "Building all services..."
	@for svc in services/*/cmd; do \
		echo "  Building $$svc..."; \
		go build ./$$svc; \
	done

# Run all tests
test:
	go test ./...

# Lint all Go code
lint:
	golangci-lint run ./...

# Start development environment
dev:
	bash scripts/dev.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@find . -type f -name '*.exe' -delete
	@find . -type f -name '*.out' -delete
	@find . -type d -name '__debug_bin' -exec rm -rf {} + 2>/dev/null || true
	@echo "Done."