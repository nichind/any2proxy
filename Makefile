.PHONY: fmt test lint build proto

fmt:
	gofmt -w ./cmd ./internal ./tests

test:
	go test ./...

lint:
	go vet ./...

build:
	go build ./cmd/a2p

proto:
	buf generate
