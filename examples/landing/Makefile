SHELL := /bin/bash

APP := bin/landing

.PHONY: dev build vendor-assets css generate test

dev:
	templ generate ./...
	bun run vendor:assets
	bun run build:css
	go run ./cmd/server

build:
	templ generate ./...
	bun run vendor:assets
	bun run build:css
	go build -o $(APP) ./cmd/server

vendor-assets:
	bun run vendor:assets

css:
	bun run build:css

generate:
	templ generate ./...

test:
	go test ./...
