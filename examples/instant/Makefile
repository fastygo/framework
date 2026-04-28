SHELL := /bin/bash

APP := bin/instant

.PHONY: dev build test bench

dev:
	go run ./cmd/server

build:
	go build -o $(APP) ./cmd/server

test:
	go test ./...

bench:
	go test ./internal/site -bench=. -benchmem
