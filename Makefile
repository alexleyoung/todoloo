.PHONY: build run start stop test lint clean install

build:
	go build -ldflags="-s -w" -o bin/todoloo ./cmd/server
	go build -ldflags="-s -w" -o bin/tdl ./cmd/tdl

install: build
	install -d $(HOME)/bin
	install bin/todoloo $(HOME)/bin/
	install bin/tdl $(HOME)/bin/

start:
	./bin/todoloo start &

stop:
	./bin/todoloo stop

run:
	go run ./cmd/server run

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
