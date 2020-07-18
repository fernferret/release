all: build

git_hash = $(shell git describe --dirty --always)

build:
	go build -ldflags "-X main.version=$(git_hash)" -o bin/release -v ./cmd/release

update:
	go get -u
	go mod tidy
	go mod vendor

# This is awkward from a makefile, but here's how you'd do in-place runs
# run:
# 	go run ./cmd/release

test:
	go test -v ./...

fmt:
	go fmt ./...

.PHONY: test fmt update

