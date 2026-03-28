.DEFAULT_GOAL := all

BINARY := uzura
COVER_PROFILE := coverage.out
COVER_HTML := coverage.html

## Build

.PHONY: build
build: ## Build the uzura binary
	go build -o $(BINARY) ./cmd/uzura

## Quality

.PHONY: fmt
fmt: ## Format code with gofmt and goimports
	gofmt -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not found, skipping"; \
	fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (skips if not installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping"; \
	fi

.PHONY: test
test: ## Run all tests with race detector
	go test ./... -race

.PHONY: bench
bench: ## Run benchmarks
	go test ./... -bench=. -benchmem

.PHONY: quality
quality: fmt vet lint test ## Run fmt + vet + lint + test

## Coverage

.PHONY: cover
cover: ## Generate coverage report
	go test ./... -coverprofile=$(COVER_PROFILE)
	go tool cover -html=$(COVER_PROFILE) -o $(COVER_HTML)
	@echo "Coverage report: $(COVER_HTML)"

## Cleanup

.PHONY: clean
clean: ## Remove binary and generated files
	rm -f $(BINARY) $(COVER_PROFILE) $(COVER_HTML)
	go clean -cache -testcache

## Default

.PHONY: all
all: quality build ## Run quality checks and build (default)

## Help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
