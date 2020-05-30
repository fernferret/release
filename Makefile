
all: build

build:
	go build -o bin/release -v ./cmd/release

# This is awkward from a makefile, but here's how you'd do in-place runs
# run:
# 	go run ./cmd/release

test:
	go test ./... -v

fmt:
	go fmt ./... -v

.PHONY: install test fmt

