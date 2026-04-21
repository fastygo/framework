# Hello, world

This post is a practical look at how the demo blog in `examples/blog` is rendered.

There is no CMS, no database dependency for content, and no heavy client-side rendering for article conversion. Articles live in markdown files inside the repository and are transformed on the server side before the first request.

In short, you write markdown, register a slug, and get a real SSR page from the same Go application.

## How the blog render pipeline works

1. Markdown files are discovered from embedded locale directories.
2. `pkg/content-markdown` parses and renders them into HTML during startup.
3. On every request, the cached HTML is reused and inserted into templates.

This makes the runtime path simpler and more deterministic.

### Startup rendering

Rendering at startup removes repeated conversion work from request handling. That keeps response times more predictable as traffic increases.

### In-memory cache

A warm cache means opening the same post repeatedly does not repeat expensive conversion, which is useful for frequently visited pages and stable UX.

## Why this helps teams

- Content and code are versioned together in git.
- The pipeline is easy to inspect and reason about.
- The setup is lightweight to scaffold for internal tools and examples.

You can think in three steps: source markdown → rendered cache → templ view.

## Next steps

Drop a new file under `content/i18n/<locale>/<slug>.md`, register it in `internal/site/blog/feature.go`, regenerate templates if needed, and restart the server.
