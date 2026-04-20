# Contributing to `fastygo/framework`

Thank you for considering a contribution. This document is the
short, practical guide. For the bird's-eye architectural view start
with [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md); for the release
mechanics see [`RELEASE.md`](RELEASE.md).

---

## 1. The three pillars (read this first)

Every change is judged against three constraints. A patch that
violates any of them will be asked to shrink, defer the dependency,
or split.

1. **No unnecessary code.** A line earns its place by removing more
   complexity than it adds.
2. **No unnecessary requests.** No hidden DB calls, no surprise HTTP
   round-trips, no goroutine, cache, or stack leaks. Long-lived
   resources need a `Stop` / `Cleanup` / `Close` path that is
   actually wired into `App.Run`.
3. **No unnecessary external dependencies.** The framework's direct
   dependencies fit on one line: `github.com/a-h/templ`,
   `github.com/yuin/goldmark` (markdown only, opt-in via
   `pkg/content-markdown`), and `go.uber.org/goleak` (test-only).
   New direct dependencies require an ADR.

---

## 2. Repository orientation in 30 seconds

```text
pkg/                    public framework library (this is what's released)
examples/<name>/        five reference apps; each is its own Go module
docs/                   long-form documentation, including ADRs in docs/adr/
scripts/                small helper programs (no business logic)
.project/               local planning notes; gitignored on this branch
go.work                 development-time link between pkg/ and examples/
.golangci.yml           lint configuration (disable-all + a curated set)
.github/workflows/ci.yml CI: framework lint+test, then build every example
```

The framework module **never** imports anything outside its own
`pkg/...`. This invariant is enforced by
`scripts/check-no-root-imports.go`, which is part of `make ci`.

---

## 3. Local setup

### 3.1. Toolchain

| Tool                  | Version    | Why                                          |
|-----------------------|------------|----------------------------------------------|
| Go                    | 1.25.x     | matches `go.mod`; required for new analyzers (`waitgroup`, `hostport`) |
| `golangci-lint`       | v1.64.x    | matches `.golangci.yml` and the CI workflow  |
| `templ`               | v0.3.1001  | only needed if you touch examples            |
| `make`                | any GNU    | optional on Windows (`scripts/preflight.sh` is the portable fallback) |

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
go install github.com/a-h/templ/cmd/templ@v0.3.1001
```

### 3.2. Clone and verify

```bash
git clone https://github.com/fastygo/framework.git
cd framework

go mod download
make ci          # runs go test ./... + the no-root-imports check + go vet
```

If `make` is unavailable (Windows without WSL), use the portable
preflight script — it runs the same commands without GNU make:

```bash
./scripts/preflight.sh
```

To match CI's resolution mode (no workspace, like a real consumer):

```bash
PREFLIGHT_CI_PARITY=1 ./scripts/preflight.sh
```

### 3.3. Working with `go.work`

`go.work` is a development-time convenience: it links `pkg/` to
every starter under `examples/` so a change in the framework is
immediately visible in every example. CI explicitly disables this
(`GOWORK=off`) so it consumes the framework the way an external
project would.

If you see surprising "version mismatch" errors locally, run with
`GOWORK=off` to reproduce the CI environment.

---

## 4. Making a change

### 4.1. Branch

```bash
git checkout -b <topic>     # short kebab-case, e.g. fix-worker-stop-leak
```

There is no `dev` branch. PRs target `main`. Long-running
exploratory work lives on the `improved` branch by convention.

### 4.2. Code

Follow the rules in `docs/ARCHITECTURE.md`:

- Imports respect the layered model (stdlib → primitives → http →
  app). New cross-package edges need a paragraph in the PR
  description explaining why.
- Public API lives in `pkg/...`. Anything else is a starter's
  internal detail.
- All comments and identifiers are written in English.
- Comments explain **intent**, not what the code does. Skip
  obvious narration ("// increment counter").

### 4.3. Tests

A patch without tests is incomplete. Bugfixes always include a test
that fails on the previous code; new features include unit tests
for the happy path and at least one failure path.

Special test categories already in the repo — please follow the
pattern:

| Pattern                       | Purpose                                                      |
|-------------------------------|--------------------------------------------------------------|
| `*_test.go`                   | Standard unit tests                                          |
| `leak_test.go`                | `go.uber.org/goleak.VerifyTestMain` for any package that touches goroutines |
| `*_audit_test.go`             | Structured-log assertions for security-sensitive emit paths  |
| `example_test.go`             | Executable godoc examples with `// Output:` blocks           |
| `*_integration_test.go`       | Cross-package wiring tests (e.g. `tracing_integration_test.go`) |

If you add a package that spawns goroutines, add a `leak_test.go`.
The framework relies on `goleak` to keep Phase 1's "no orphan
goroutines" guarantee.

### 4.4. Docs

If your patch changes a documented contract, update the document in
the **same commit**:

- Middleware order, `Feature` interface, lifecycle of `App.Run` →
  `docs/ARCHITECTURE.md`.
- New env var or default → `docs/12-FACTOR.md` and the `Config`
  godoc.
- Wire format / metric names / log fields → `docs/OBSERVABILITY.md`.
- Any exported symbol → godoc on the symbol itself (verified by
  `go run ./scripts/godoc-audit`).

For new public API, add an executable example:

```go
func ExampleNewThing() {
    out := pkg.NewThing("hello")
    fmt.Println(out)
    // Output: hello
}
```

Examples are tests **and** documentation: they appear on pkg.go.dev
and break the build if the API changes.

---

## 5. Local quality gates

Before opening a PR, run all three:

```bash
make ci               # go test + no-root-imports + go vet
make lint-go          # golangci-lint (optional locally, mandatory in CI)
go run ./scripts/godoc-audit ./pkg/...   # zero violations expected
```

For a CI-equivalent run (matches the GitHub workflow exactly):

```bash
PREFLIGHT_CI_PARITY=1 ./scripts/preflight.sh
```

Optional but encouraged when touching concurrency code:

```bash
PREFLIGHT_RUN_RACE=1 ./scripts/preflight.sh   # adds go test -race
```

To smoke-test that every example still builds against your changes:

```bash
make examples
```

---

## 6. Commit and PR style

### 6.1. Commits

- Imperative subject, ≤72 chars (`fix(worker): drain wg before shutdown`).
- Conventional prefixes are encouraged but not enforced: `feat`,
  `fix`, `docs`, `test`, `chore`, `refactor`, `perf`, `release`.
- One logical change per commit. If a single PR contains several
  refactors, keep them in separate commits so reviewers can read
  the history.

### 6.2. PR description

Include, at minimum:

1. **Why** — the problem or the user-visible improvement.
2. **What** — the contract change in plain prose.
3. **How** — the approach if it is non-obvious.
4. **Tests** — what you added and which scenarios it covers.
5. **Docs** — which `docs/*.md` files were updated.
6. **Risk** — any known gotcha for the reviewer.

Link the corresponding entry in `.project/roadmap-framework.md` if
the change is part of a planned phase. If it introduces an
architectural decision, link the new ADR (see §8).

---

## 7. CHANGELOG

Update `CHANGELOG.md` in the same PR.

- Add an entry under `## [Unreleased]` in the appropriate subsection
  (`Added`, `Changed`, `Removed`, `Security`, `Fixed`, `Deprecated`).
- Speak in user terms: what changes for someone importing
  `pkg/...`. "Refactored internal foo" is rarely interesting; "new
  `Foo.Stop(ctx) error` method, drains the queue before returning"
  is.
- Bug fixes link the issue or describe the failure mode in one
  sentence so downstream consumers can decide if they were affected.

The `[Unreleased]` section is moved into a dated version on release
day — see `RELEASE.md`.

---

## 8. Architectural Decision Records

Open an ADR when your change:

- introduces a new direct dependency,
- changes the lifecycle of `App.Run`,
- changes the order of middleware,
- adds or removes a `pkg/...` subdirectory,
- changes a security default (cookies, headers, body limits, rate
  limits, antibot),
- adds a public extension point (new optional `Feature` interface).

ADRs live in `docs/adr/NNNN-short-slug.md`. Use the existing files
as the template; the canonical structure is:

```text
- Status: Proposed | Accepted | Superseded by NNNN | Rejected
- Date: YYYY-MM-DD
- Deciders: list of GitHub handles
- Release: vX.Y.Z (when known)

## Context
## Decision
## Consequences
## Alternatives considered
```

Existing records: `0001` (split framework/examples), `0002` (Phase
1 leak hardening), `0003` (Phase 2 observability). They are short
on purpose: an ADR fits in one screen.

---

## 9. CI overview

`.github/workflows/ci.yml` runs two jobs on every PR:

1. **`framework`** — `make ci`, `go vet`, `golangci-lint` with
   `GOWORK=off`. This is the gate.
2. **`examples`** — matrix over `[landing, web, blog, docs,
   dashboard]`. Generates `templ` files and runs `go build ./...`
   for each starter. Catches API breaks the unit tests miss.

A red `examples` job almost always means "you changed a public
signature in `pkg/`; please update the starters or add a deprecation
shim."

---

## 10. Releases

Releases are cut by maintainers from `main` following
[`RELEASE.md`](RELEASE.md). Contributors do **not** tag versions
themselves; instead, make sure the `[Unreleased]` section of
`CHANGELOG.md` is informative — that block becomes the GitHub
release notes verbatim.

---

## 11. Reporting security issues

Please **do not open public issues for vulnerabilities.** Email the
maintainers privately so we can prepare a coordinated fix and a
CVE if warranted. The full threat model lives in
`docs/SECURITY.md`.

---

## 12. License and provenance

By contributing you agree that your code will be released under the
project's license (MIT unless the repository says otherwise) and
that you have the right to submit it. There is no separate CLA: the
commit history is the record.

Welcome aboard.
