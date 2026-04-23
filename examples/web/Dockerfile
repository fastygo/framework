# When this example is split into its own repository, drop the framework
# COPY layer (lines marked "monorepo only") and replace it with a regular
# `go mod download` against a tagged release of the framework.
#
# Build it from the framework repository root:
#   docker build -f examples/web/Dockerfile -t fastygo-example-web .
FROM golang:1.25-bookworm AS build

WORKDIR /src

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates bash unzip \
    && rm -rf /var/lib/apt/lists/*

ENV BUN_INSTALL=/root/.bun
ENV PATH="${BUN_INSTALL}/bin:${PATH}"

RUN curl -fsSL https://bun.sh/install | bash

# Monorepo only: copy the local framework module so the example's
# `replace github.com/fastygo/framework => ../..` directive resolves.
COPY go.mod go.sum ./framework/
COPY pkg ./framework/pkg
COPY scripts ./framework/scripts

WORKDIR /src/app
COPY examples/web/package.json ./
RUN bun install

COPY examples/web/go.mod examples/web/go.sum* ./
COPY examples/web/internal ./internal
COPY examples/web/cmd ./cmd
COPY examples/web/web ./web

# Adjust the replace path so it resolves the framework copy above.
RUN sed -i 's|replace github.com/fastygo/framework => ../..|replace github.com/fastygo/framework => /src/framework|' go.mod
RUN go mod download
RUN bun run vendor:assets
RUN bun run build:css
RUN go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate ./...
RUN go build -ldflags="-s -w" -o /out/web ./cmd/server

FROM debian:bookworm-slim AS runtime
WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/web /app/web
COPY --from=build /src/app/web/static /app/static

ENV APP_BIND=0.0.0.0:80
ENV APP_STATIC_DIR=static
ENV APP_DEFAULT_LOCALE=en
ENV APP_AVAILABLE_LOCALES=en,ru

EXPOSE 80
CMD ["./web"]
