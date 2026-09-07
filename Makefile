.PHONY: help build install test lint fmt check setup clean run format analyze

BINARY := sci

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

build: ## Compile ./sci
	go build -o $(BINARY) ./cmd/sci

install: ## go install into $GOBIN
	go install ./cmd/sci

test: ## Run tests with the race detector
	go test -race ./...

lint: ## Run the whole gate — every hook, every file
	pre-commit run --all-files

fmt: ## Rewrite with gofmt
	gofmt -w .

setup: ## Install the pre-commit hook
	pre-commit install

check: ## Run all pre-commit checks on the whole tree
	pre-commit run --all-files

clean: ## Remove the binary
	rm -f $(BINARY)

run: ## Run the binary
	go run ./cmd/sci $(ARGS)

format: ## Rewrite the sources to gofmt form
	gofmt -w .

analyze: ## Lint with the house rule set
	golangci-lint run ./...
