SHELL := /bin/bash

APP := bin/blog

.PHONY: dev build vendor-assets css generate test layout-audit

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

layout-audit:
	(cd ../.. && go run ./scripts/layout-audit ./examples/blog/internal/site/views)
