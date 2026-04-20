# Build it from the framework repository root:
#   docker build -f examples/docs/Dockerfile -t fastygo-example-docs .
FROM golang:1.25-bookworm AS build

WORKDIR /src

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates gnupg bash \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./framework/
COPY pkg ./framework/pkg
COPY scripts ./framework/scripts

WORKDIR /src/app
COPY examples/docs/package.json examples/docs/package-lock.json* ./
RUN npm install

COPY examples/docs/go.mod examples/docs/go.sum* ./
COPY examples/docs/internal ./internal
COPY examples/docs/content ./content
COPY examples/docs/cmd ./cmd
COPY examples/docs/web ./web

RUN sed -i 's|replace github.com/fastygo/framework => ../..|replace github.com/fastygo/framework => /src/framework|' go.mod
RUN go mod download
RUN bash /src/framework/scripts/sync-ui8kit-css.sh web/static
RUN npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/app.css --minify
RUN go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate ./...
RUN go build -ldflags="-s -w" -o /out/docs ./cmd/server

FROM debian:bookworm-slim AS runtime
WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/docs /app/docs
COPY --from=build /src/app/web/static /app/static

ENV APP_BIND=0.0.0.0:80
ENV APP_STATIC_DIR=static
ENV APP_DEFAULT_LOCALE=en
ENV APP_AVAILABLE_LOCALES=en,ru

EXPOSE 80
CMD ["./docs"]
