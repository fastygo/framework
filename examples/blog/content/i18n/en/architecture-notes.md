# Architecture notes

This note explains how the blog example is put together and why the pieces are connected this way.

`examples/blog` demonstrates a compact but complete pattern: locale-aware routes, cached rendering, and templated SSR output.

## Core components

1. `pkg/content-markdown` — markdown-to-HTML conversion layer.
2. `pkg/web/locale` — request locale resolution.
3. `pkg/cache` — memoization layer for already-rendered HTML.

These combine into a predictable pipeline from content to response.

## How a post page is produced

`internal/site/blog/feature.go` defines available slugs and routes such as `/` and `/posts/{slug}`.

For each request:

- resolve locale and slug,
- select the prepared HTML for this pair,
- pass view data into templ,
- return an SSR response.

The expensive work already happened at startup, so requests stay lightweight.

## How to add a new post

Typical diff:

1. Add `content/i18n/<locale>/<slug>.md`.
2. Register it in `internal/site/blog/feature.go`.
3. Rebuild templates and restart if needed.

## Note for future growth

Keep markdown structure clear with meaningful paragraphs and consistent headings; this improves readability for users and extraction quality for browser reader modes.
