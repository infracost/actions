default: help

.PHONY: help
help: ## Show this help.
	@fgrep -h "##" $(MAKEFILE_LIST)  | fgrep -v fgrep | sed -e 's/:.*##/:##/' | awk -F':##' '{printf "%-12s %s\n",$$1, $$2}'

.PHONY: build
build: ## Build the project
	go build -o bin/scanner main.go

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: lint_install
lint_install: ## Install golangci-lint
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: lint
lint: lint_install ## Run linting operations
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Check formatting
	@gofmt -l . | while read -r f; do \
		echo "The following files are not formatted correctly:"; \
		gofmt -l .; \
		exit 1; \
	done