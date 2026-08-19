.DEFAULT_GOAL := help

BINARY := se2calc

.PHONY: help build install test cover fmt vet check tidy clean

help: ## List available targets
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the se2calc binary
	go build -o $(BINARY) .

install: ## Build and install se2calc with go install
	go install .

test: ## Run the full test suite
	go test ./...

cover: ## Run tests with a coverage summary
	go test -cover ./...

fmt: ## Format all Go files in place
	gofmt -l -w .

vet: ## Run go vet
	go vet ./...

check: ## CI-style gate: formatting, vet, tests
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then \
		echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go test ./...

tidy: ## Sync go.mod/go.sum with imports
	go mod tidy

clean: ## Remove the built binary
	rm -f $(BINARY)
