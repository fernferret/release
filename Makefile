
all: build

build:
	go build -o bin/release -v ./cmd/release

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

