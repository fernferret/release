
all: build

build:
	go build -o bin/release -v ./cmd/release

test:
	go test ./... -v

fmt:
	go fmt ./... -v

.PHONY: install test fmt

