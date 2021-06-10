# Go parameters
#= reminders
#= GOFLAGS="-count=1" ---> turn off test caching
#= -covermode=count   ---> how many times did each statement run?
#=			=atomic   ---> like count, but counts precisely in parallel programs
GOENV=CGO_ENABLED=0 GOFLAGS="-count=1"
GOCMD=$(GOENV) go
GOGET=go get
GOTEST=$(GOCMD) test -covermode=atomic -coverprofile=./coverage.out -v -timeout=20m

.EXPORT_ALL_VARIABLES:
APP_PORT?=8080
DB_HOST?=localhost
DB_PORT?=5555
DB_NAME?=test
DB_USER?=test
DB_PASSWORD?=test
# CACHE_HOST?=localhost
# CACHE_PORT?=6679

.PHONY: pg, stop-pg, restart-pg, redis, stop-redis, restart-redis, restart-all, stop-all
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
	@timeout 90s bash -c 'until docker exec urlpg pg_isready ; do sleep 1 ; done'

restart-pg: stop-pg
restart-pg: pg

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

stop-all: stop-pg
stop-all: stop-redis

restart-all: restart-pg
restart-all: restart-redis

.PHONY: unittest, e2e, alltest, see-coverage
unittest:
	@${GOTEST} `go list ./... | grep -v /e2e`

# TODO: using `restart-all` after supporting redis cache engine
e2e: restart-pg
e2e:
	@${GOTEST} `go list ./... | grep /e2e`

alltest: restart-all
alltest:
	@${GOTEST} ./...

see-coverage:
	@go tool cover -html=coverage.out

.PHONY: run
run:
	@${GOCMD} run main.go


.PHONY: tidy
tidy:
	go mod tidy