.PHONY: build test test-race coverage fmt vet

build:
	go build ./...

test:
	go test ./...

test-race:
	go test ./... -race

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

fmt:
	gofmt -w .

vet:
	go vet ./...
