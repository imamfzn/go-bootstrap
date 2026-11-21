# Loads .env when present, so `make run-api` works after `cp .env.example .env`.
ifneq (,$(wildcard .env))
include .env
export
endif

MIGRATIONS := db/migrations
GOOSE      := go tool -modfile=tools.mod goose -dir $(MIGRATIONS)
PUBSUB     := http://$(PUBSUB_EMULATOR_HOST)/v1/projects/$(PUBSUB_PROJECT_ID)
IMAGE      ?= go-bootstrap

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

## --- Setup -----------------------------------------------------------------

.PHONY: rename
rename: ## Rename the module after cloning the template: make rename module=github.com/acme/orders
	@test -n "$(module)" || (echo "usage: make rename module=github.com/acme/orders" && exit 1)
	@old=$$(go list -m) && \
	  go mod edit -module $(module) && \
	  grep -rl "$$old" --include='*.go' --include='*.mod' --include='*.yaml' --include='*.md' . | xargs perl -pi -e "s|\Q$$old\E|$(module)|g" && \
	  echo "module renamed from $$old to $(module)"

.PHONY: up
up: ## Start Postgres and the Pub/Sub emulator, then create topics
	docker compose up -d --wait postgres
	docker compose up -d pubsub
	@$(MAKE) --no-print-directory pubsub-setup

.PHONY: down
down: ## Stop local dependencies
	docker compose down

.PHONY: pubsub-setup
pubsub-setup: ## Create the topic and subscription in the emulator
	@until curl -sf -o /dev/null $(PUBSUB)/topics; do sleep 1; done
	@curl -s -o /dev/null -X PUT $(PUBSUB)/topics/$(TOPIC_ORDER_CREATED)
	@curl -s -o /dev/null -X PUT $(PUBSUB)/subscriptions/$(SUBSCRIPTION_ORDER_CREATED) \
	  -H 'Content-Type: application/json' \
	  -d '{"topic": "projects/$(PUBSUB_PROJECT_ID)/topics/$(TOPIC_ORDER_CREATED)"}'
	@echo "pubsub emulator: topic $(TOPIC_ORDER_CREATED), subscription $(SUBSCRIPTION_ORDER_CREATED)"

## --- Run -------------------------------------------------------------------

.PHONY: run-api
run-api: ## Run the API
	go run ./cmd/api

.PHONY: run-worker
run-worker: ## Run the worker
	go run ./cmd/worker

.PHONY: run-job
run-job: ## Run a job: make run-job name=expire-orders
	@test -n "$(name)" || (echo "usage: make run-job name=expire-orders" && exit 1)
	go run ./cmd/job $(name)

## --- Quality ---------------------------------------------------------------

.PHONY: test
test: ## Run the tests (repository tests need TEST_DATABASE_URL and `make migrate-up`)
	go test -race ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (install: https://golangci-lint.run/welcome/install)
	golangci-lint run

.PHONY: fix
fix: ## Apply the Go modernizers and format
	go fix ./...
	gofmt -w .

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	go mod tidy

.PHONY: check
check: tidy vet lint test ## Everything CI runs

## --- Migrations ------------------------------------------------------------

.PHONY: migrate-new
migrate-new: ## Create a migration: make migrate-new name=add_orders_index
	@test -n "$(name)" || (echo "usage: make migrate-new name=add_orders_index" && exit 1)
	$(GOOSE) create $(name) sql

.PHONY: migrate-up
migrate-up: ## Apply pending migrations
	$(GOOSE) postgres "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration (development only)
	$(GOOSE) postgres "$(DATABASE_URL)" down

.PHONY: migrate-status
migrate-status: ## Show which migrations are applied
	$(GOOSE) postgres "$(DATABASE_URL)" status

## --- Build -----------------------------------------------------------------

.PHONY: build
build: ## Build the three binaries into ./bin
	CGO_ENABLED=0 go build -trimpath -o bin/ ./cmd/...

.PHONY: image
image: ## Build the container image
	docker build -f deploy/Dockerfile -t $(IMAGE) .
