GOENV=CGO_ENABLED=0 GOFLAGS="-count=1"
GOCMD=$(GOENV) go
GOTEST=$(GOCMD) test -covermode=atomic -coverprofile=./coverage.out -v -timeout=20m
GOBENCH=$(GOCMD) test -bench=. -benchmem -run=none .

test:
	@${GOTEST} ./...

see-coverage:
	@go tool cover -html=coverage.out

bench:
	@${GOBENCH}
