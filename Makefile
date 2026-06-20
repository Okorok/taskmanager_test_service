PROJECT_NAME := taskmanager

DOCKER_HOST_ENDPOINT := $(shell docker context inspect -f '{{.Endpoints.docker.Host}}' 2>/dev/null)
INTEGRATION_ENV := DOCKER_HOST=$(DOCKER_HOST_ENDPOINT) TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock

.PHONY: fmt lint test test-unit test-integration docker-up docker-down generate

generate:
	mockery

fmt:
	go fmt ./...

lint:
	golangci-lint run ./... --fix

docker-up:
	docker compose down -v || true
	docker compose up --build

docker-down:
	docker compose down -v

test-unit:
	go test ./...

test-integration:
	$(INTEGRATION_ENV) go test -tags=integration -count=1 ./...

test: test-unit test-integration
