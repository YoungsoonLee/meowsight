.PHONY: all build clean test lint fmt run-api run-proxy run-ingest run-worker infra infra-down

BINARY_DIR := bin
GO := go
GOFLAGS := -trimpath

BINARIES := meowsight-api meowsight-proxy meowsight-ingest meowsight-worker meowctl

all: build

## build: Build all binaries
build: $(BINARIES)

$(BINARIES):
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$@ ./cmd/$@

## clean: Remove build artifacts
clean:
	rm -rf $(BINARY_DIR)

## test: Run all tests
test:
	$(GO) test ./... -race -cover

## lint: Run linter
lint:
	$(GO) vet ./...
	staticcheck ./...

## fmt: Format code
fmt:
	$(GO) fmt ./...
	goimports -w .

## run-api: Run API server
run-api: build
	./$(BINARY_DIR)/meowsight-api

## run-proxy: Run LLM proxy
run-proxy: build
	./$(BINARY_DIR)/meowsight-proxy

## run-ingest: Run ingestion worker
run-ingest: build
	./$(BINARY_DIR)/meowsight-ingest

## run-worker: Run background worker
run-worker: build
	./$(BINARY_DIR)/meowsight-worker

## infra: Start local infrastructure
infra:
	docker compose up -d

## infra-down: Stop local infrastructure
infra-down:
	docker compose down

## infra-reset: Stop and remove all data
infra-reset:
	docker compose down -v

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
