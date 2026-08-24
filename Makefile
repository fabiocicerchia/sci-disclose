.PHONY: help build install test lint fmt check setup clean

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

lint: ## go vet + gofmt check
	go vet ./...
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

fmt: ## Rewrite with gofmt
	gofmt -w .

setup: ## Install the pre-commit hook
	pre-commit install

check: ## Run all pre-commit checks on the whole tree
	pre-commit run --all-files

clean: ## Remove the binary
	rm -f $(BINARY)
