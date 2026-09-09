.PHONY: help build install test lint fmt check setup clean run format analyze

BINARY := sci

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

build: ## Compile ./sci
	go build -o $(BINARY) ./cmd/sci

## install: install the binary and its man page; PREFIX=/usr/local for a system path
install:
ifeq ($(strip $(PREFIX)),)
	go install ./cmd/sci
	install -d "$(USER_MANDIR)"
	install -m 0644 man/$(BINARY).1 "$(USER_MANDIR)/$(BINARY).1"
	@dir="$$(go env GOBIN)"; [ -n "$$dir" ] || dir="$$(go env GOPATH)/bin"; \
		echo "installed $$dir/$(BINARY) and $(USER_MANDIR)/$(BINARY).1"
else
	@$(MAKE) build
	install -d "$(DESTDIR)$(PREFIX)/bin" "$(DESTDIR)$(PREFIX)/share/man/man1"
	install -m 0755 $(BINARY) "$(DESTDIR)$(PREFIX)/bin/$(BINARY)"
	install -m 0644 man/$(BINARY).1 "$(DESTDIR)$(PREFIX)/share/man/man1/$(BINARY).1"
	@echo "installed $(DESTDIR)$(PREFIX)/bin/$(BINARY)"
endif

## uninstall: remove what `make install` put down
uninstall:
ifeq ($(strip $(PREFIX)),)
	@dir="$$(go env GOBIN)"; [ -n "$$dir" ] || dir="$$(go env GOPATH)/bin"; \
		rm -f "$$dir/$(BINARY)" "$(USER_MANDIR)/$(BINARY).1"
else
	rm -f "$(DESTDIR)$(PREFIX)/bin/$(BINARY)" \
		"$(DESTDIR)$(PREFIX)/share/man/man1/$(BINARY).1"
endif

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
