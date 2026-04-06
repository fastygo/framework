SHELL := /bin/bash

APP := bin/framework
PKG := ./cmd/server
DOCS_PKG := ./cmd/docs
NO_ROOT_IMPORT_CHECK := go run ./scripts/check-no-root-imports.go

dev:
	templ generate
	npm run build:css
	go run $(PKG)

build:
	templ generate
	npm run build:css
	go build -o $(APP) $(PKG)

dev-docs:
	templ generate
	npm run build:docs:css
	go run $(DOCS_PKG)

build-docs:
	templ generate
	npm run build:docs:css
	go build -o bin/docs $(DOCS_PKG)

css-dev:
	npm run dev:css

css-build:
	npm run build:css

test:
	go test ./...

lint:
	go test ./...
	$(NO_ROOT_IMPORT_CHECK)

lint-ci:
	$(MAKE) lint

ci:
	$(MAKE) lint-ci

lint-bash:
	./scripts/check-no-root-imports.sh

generate:
	templ generate
