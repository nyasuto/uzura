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

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: ## Run all tests with race detector
	go test ./... -race

.PHONY: bench
bench: ## Run benchmarks
	go test ./... -bench=. -benchmem

.PHONY: quality
quality: fmt lint test ## Run fmt + lint + test

## Coverage

.PHONY: cover
cover: ## Generate coverage report
	go test ./... -coverprofile=$(COVER_PROFILE)
	go tool cover -html=$(COVER_PROFILE) -o $(COVER_HTML)
	@echo "Coverage report: $(COVER_HTML)"

## E2E tests

.PHONY: e2e
e2e: build ## Run E2E tests (requires Node.js + npm install in e2e/)
	@if [ ! -d e2e/node_modules ]; then \
		echo "Installing e2e dependencies..."; \
		cd e2e && npm install --silent; \
	fi
	go test -tags e2e -timeout 60s ./e2e/...

## WPT

.PHONY: wpt-fetch
wpt-fetch: ## Download WPT test suite (sparse checkout)
	bash scripts/wpt-fetch.sh

.PHONY: wpt
wpt: build ## Run WPT tests (downloads WPT if needed)
	@if [ ! -d testdata/wpt/.git ]; then \
		echo "WPT not found, downloading..."; \
		bash scripts/wpt-fetch.sh; \
	fi
	./$(BINARY) wpt $(WPT_DIRS)

## Cleanup

.PHONY: clean
clean: ## Remove binary and generated files
	rm -f $(BINARY) $(COVER_PROFILE) $(COVER_HTML)
	go clean -cache -testcache

## Git hooks

.PHONY: install-hooks
install-hooks: ## Install pre-commit hook that runs make quality
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make quality' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "pre-commit hook installed"

## Default

.PHONY: all
all: quality build ## Run quality checks and build (default)

## Help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
