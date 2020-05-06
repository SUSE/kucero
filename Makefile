VERSION=$(shell git symbolic-ref --short HEAD)-$(shell git rev-parse --short HEAD)

all: test build

verify:
	go mod tidy
	go mod verify

test: verify
	go test -count=1 ./...

build: verify
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION)" -o cmd/kucero/kucero cmd/kucero/main.go

clean:
	go clean -x -i ./...

.PHONY: all build clean
