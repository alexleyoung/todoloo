.PHONY: build run test lint clean

build:
	go build -ldflags="-s -w" -o bin/todoloo ./cmd/server

run:
	go run ./cmd/server --config config.yaml

test:
	go test ./... -race -count=1

lint:
	golangci-lint run

clean:
	rm -rf bin/

build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/todoloo-linux ./cmd/server

build-mac-arm:
	GOOS=darwin GOARCH=arm64 go build -o bin/todoloo-mac ./cmd/server
