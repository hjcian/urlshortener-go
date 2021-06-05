# Go parameters
GOENV=CGO_ENABLED=0 GOFLAGS="-count=1"
GOCMD=$(GOENV) go
GOGET=go get
GOTEST=$(GOCMD) test -covermode=atomic -coverprofile=./coverage.out -v -timeout=20m

.EXPORT_ALL_VARIABLES:
APP_PORT?=8080
CACHE_PORT?=6679
DB_PORT?=5555
DB_NAME?=test
DB_USER?=test
DB_PASSWORD?=test

.PHONY: pg, stop-pg, restart-pg
stop-pg:
	@echo "[`date`] Stopping previous launched postgres [if any]"
	docker stop urlpg || true

pg:
	@echo "[`date`] Starting Postgres container"
	docker run -d --rm --name urlpg \
		-p ${DB_PORT}:5432 \
		-e POSTGRES_DB=${DB_NAME} \
		-e POSTGRES_USER=${DB_USER} \
		-e POSTGRES_PASSWORD=${DB_PASSWORD} \
		postgres:12

restart-pg: stop-pg
restart-pg: pg

.PHONY: redis, stop-redis, restart-redis
stop-redis:
	@echo "[`date`] Stopping previous launched redis [if any]"
	docker stop urlredis || true

redis:
	@echo "[`date`] Starting redis container"
	docker run -d --rm --name urlredis \
		-p ${CACHE_PORT}:6379 \
		redis:4.0-alpine

restart-redis: stop-redis
restart-redis: redis

.PHONY: run
run:
	@${GOCMD} run main.go

.PHONY: test
test:
	@${GOTEST} ./...

.PHONY: see-coverage
see-coverage:
	@go tool cover -html=coverage.out


