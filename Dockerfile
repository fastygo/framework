FROM golang:1.25.5-bookworm AS build

WORKDIR /src

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates gnupg bash \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum package.json package-lock.json ./
COPY scripts ./scripts

RUN npm ci
RUN go mod download
RUN go mod download github.com/fastygo/ui8kit@v0.2.5

COPY . .

RUN npm run sync:ui8kit
RUN npm run build:css
RUN go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
RUN go build -ldflags="-s -w" -o /out/framework ./cmd/server

FROM debian:bookworm-slim AS runtime

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/framework /app/framework
COPY --from=build /src/internal/site/web/static /app/static

ENV APP_BIND=0.0.0.0:80
ENV APP_STATIC_DIR=static
ENV APP_DEFAULT_LOCALE=en
ENV APP_AVAILABLE_LOCALES=en,ru
ENV APP_DATA_SOURCE=fixture
ENV OIDC_ISSUER=""
ENV OIDC_CLIENT_ID=""
ENV OIDC_CLIENT_SECRET=""
ENV OIDC_REDIRECT_URI=""
ENV SESSION_KEY=""

EXPOSE 80

CMD ["./framework"]
