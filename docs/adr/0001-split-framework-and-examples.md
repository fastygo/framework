# ADR 0001 — Split framework and examples into independent Go modules

- Status: Accepted
- Date: 2026-04-19 (retrospective; the split landed before this ADR was written)
- Deciders: framework maintainers

## Context

Through `v0.0.x` the repository hosted a single Go module that mixed three
concerns:

1. The reusable framework code under `pkg/`.
2. A demo application (welcome page, OIDC cabinet, docs site) under
   `internal/site/...` and `internal/infra/features/...`.
3. The deployment artefacts of that demo application (`Dockerfile`,
   `docker-compose.yml`, `package.json`, Tailwind pipeline, fonts).

Anyone cloning the repo to start their own project inherited:

- demo features they had to delete before writing their own code;
- a hard `require github.com/fastygo/ui8kit` dependency in the framework
  module's `go.mod` even when their UI-kit choice was different;
- a single `cmd/server/main.go` they had to rewrite.

Updates to the framework and updates to the demo site shared one
release cadence. There was no clean way for downstream projects to pull
"only the framework" through `go get`.

A brain-storm session (`.project/brain-storm.md`) explored three
options: shared monorepo with build tags, multiple small framework
"cores" per product type, or a clean library + starters split. Option
three matched real-world Go ecosystem norms (chi, fiber, gin all ship
as pure libraries with separate example repos).

## Decision

The framework is now a **pure Go library module** containing only
`pkg/...`. Every prior demo site became its own Go module under
`examples/<starter>/` with its own `go.mod`, `cmd/server/main.go`, CSS
pipeline, and `Dockerfile`. Five starters ship today:

- `examples/landing` — minimal one-page composition root
- `examples/web` — i18n marketing site with optional OIDC cabinet
- `examples/blog` — markdown-driven blog
- `examples/docs` — localized documentation site
- `examples/dashboard` — sidebar shell with auth and contacts CRUD

A top-level `go.work` resolves the local framework module during
monorepo development; consumers extracting an example to its own
repository delete the `replace` directive and depend on a tagged
framework release.

Boundary enforcement: `scripts/check-no-root-imports.go` runs in CI and
fails any framework code that imports outside `pkg/`. The check is part
of the `framework` job in `.github/workflows/ci.yml`.

Pieces that moved out of `pkg/` during the split:

- `cmd/`, `internal/`, root `Dockerfile`, `docker-compose.yml`,
  `.env.example`, `package.json`, root `pkg/web/page.go`
- The hard dependency on `github.com/fastygo/ui8kit`

Pieces added to `pkg/` so starters do not duplicate them:

- `pkg/auth` (HMAC cookie sessions + OIDC client) — extracted from the
  demo cabinet
- `pkg/web/content` (markdown rendering via goldmark)
- `pkg/web/i18n` (generic typed JSON loader over `embed.FS`)
- `pkg/web/locale` (request locale negotiator)
- `pkg/web/view` (UI-agnostic view-model structs)

`Feature` gained four optional interfaces handled by `AppBuilder` via
type assertion: `Initializer`, `Closer`, `HealthChecker`,
`BackgroundProvider`.

## Consequences

Positive:

- Downstream projects do `require github.com/fastygo/framework v0.x.y`
  and inherit nothing they did not import.
- Framework releases are independent of any specific demo site.
- The boundary is mechanical, not stylistic — CI rejects violations.
- Brain-storm scenario "four developers, four different products" is
  satisfied by five starters covering landing/marketing/blog/docs/
  dashboard.

Negative / open:

- `pkg/web/content` pulls `github.com/yuin/goldmark`, the only external
  dependency of the framework. ADR 0003 (deferred) will decide whether
  to extract it into `github.com/fastygo/content`.
- Examples carry their own `Dockerfile`/`Makefile`/`package.json`,
  which means cross-example fixes require touching N files.
- `pkg/realtime` and `pkg/jobs` from the brain-storm are not yet
  delivered; they appear in the Phase 4 roadmap.

## Alternatives considered

- **Shared monorepo with build tags** (`//go:build enterprise`).
  Rejected: tags fragment CI, hide compile errors until a tag flip,
  and Go consumers cannot select features through `go get`.
- **Three separate framework cores** (one per product type).
  Rejected: massive code duplication for transport/security/render
  with marginal payoff. Different products need different
  *composition*, not different *transport cores*.
- **Plugins via `plugin.Open`**. Rejected: Go plugins are runtime-only,
  Linux-leaning, and forfeit static analysis.

## References

- `.project/brain-storm.md` — original mozg-shturm
- `.project/compare-branch-main.md` — onboarding diff
- `.project/compare-plan-refactor.md` — plan vs reality
