# Local .env (if present) is loaded and exported to every recipe.
-include .env
export

.PHONY: help build test lint fmt vuln check run clean

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

build: ## Build the binary into bin/
	go build -o bin/embedding ./cmd/embedding

test: ## Unit tests with the race detector
	go test ./... -race -count=1

lint: ## golangci-lint (includes gofmt/goimports checks)
	golangci-lint run ./...

fmt: ## Format code
	golangci-lint fmt ./...

vuln: ## Scan dependencies for known vulnerabilities
	govulncheck ./...

check: lint test vuln ## Everything /push_github runs
	go mod tidy -diff

run: ## Run the service locally
	go run ./cmd/embedding

clean: ## Remove build output
	rm -rf bin/ coverage.out
