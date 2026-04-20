SHELL := /bin/bash

APP := bin/dashboard

.PHONY: dev build sync-ui8kit css generate test

dev:
	templ generate ./...
	npm run build:css
	go run ./cmd/server

build:
	templ generate ./...
	npm run build:css
	go build -o $(APP) ./cmd/server

sync-ui8kit:
	bash ../../scripts/sync-ui8kit-css.sh web/static

css:
	npm run build:css

generate:
	templ generate ./...

test:
	go test ./...
