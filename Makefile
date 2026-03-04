.PHONY: build run test lint clean

build:
	go build -ldflags="-s -w" -o bin/todoloo ./cmd/server
	go build -ldflags="-s -w" -o bin/tdl ./cmd/tdl

run:
	go run ./cmd/server

test:
	go test ./... -race -count=1

lint:
	golangci-lint run

clean:
	rm -rf bin/

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/todoloo-linux ./cmd/server
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/tdl-linux ./cmd/tdl

build-mac-arm:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/todoloo-mac ./cmd/server
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/tdl-mac ./cmd/tdl
