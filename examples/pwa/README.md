# FastyGo PWA Example

A TODO-style Progressive Web App example that demonstrates an installable app
shell with a manifest, root-scoped service worker, static asset caching, and an
offline navigation fallback.

The product concept is a small task app with a subscription funnel:

- dashboard with today's tasks and reminder state;
- onboarding screens;
- premium paywall and pricing;
- mock payment method/card screens;
- payment success;
- subscription management and cancellation;
- offline fallback.

Payments, authentication, and persistence are mocked. This example focuses on
PWA browser behavior and a UI8Kit app shell.

## Run

```bash
bun install
bun run vendor:assets
bun run build:css
templ generate ./...
go run ./cmd/server
```

By default the server listens on `127.0.0.1:8080`.

## Build

```bash
bun run build
go build ./...
```

## PWA Checks

After starting the app, verify:

- `/manifest.webmanifest` returns the manifest.
- `/sw.js` returns the service worker with root scope.
- `/offline` renders the offline fallback page.
- Chrome DevTools Application panel shows the service worker and cache entries.
- Navigating while offline falls back to `/offline`.

## Notes

This is intentionally not an `instant` example. It uses CSS, JS, vendored
UI8Kit assets, a manifest, service worker, and icons because the goal is an
installable/offline app shell.
