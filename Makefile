SHELL := /bin/bash

APP := bin/framework
PKG := ./cmd/server

dev:
	templ generate
	npm run build:css
	go run $(PKG)

build:
	templ generate
	npm run build:css
	go build -o $(APP) $(PKG)

css-dev:
	npm run dev:css

css-build:
	npm run build:css

test:
	go test ./...

lint:
	go test ./...

generate:
	templ generate
