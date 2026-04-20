# Release checklist

The framework is a normal Go module published via Git tags.

## Prerequisites

- All Phase PRs merged into `main` (or the active release branch).
- `GOWORK=off make ci` is green locally.
- `make examples` is green (every starter still compiles).
- `CHANGELOG.md` `[Unreleased]` section moved into a new dated entry.
- A new ADR exists under `docs/adr/` if architectural decisions changed.

## Steps

1. Bump `CHANGELOG.md`:
   - Replace `[Unreleased]` with the next version and today's date.
   - Add a fresh `[Unreleased]` placeholder near the top.
2. Commit the changelog bump:
   ```bash
   git add CHANGELOG.md
   git commit -m "release: vX.Y.Z"
   ```
3. Tag the commit:
   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin main --tags
   ```
4. Wait for the `framework` and `examples` CI jobs to pass on the tag.
5. Create a GitHub release whose body is the new `[X.Y.Z]` section
   from `CHANGELOG.md`. Mark it as the latest release.
6. Update consumers' `require github.com/fastygo/framework vX.Y.Z`
   line in any starter you maintain outside the monorepo.

## Versioning

The framework follows [Semantic Versioning](https://semver.org/):

- `0.x` releases may contain breaking changes between minor versions
  but try to stay additive within a release cycle.
- Public API surface is everything exported from `pkg/...`. Anything
  not in `pkg/` is internal to a starter.
- `1.0.0` is gated on Phase 5 of `.project/roadmap-framework.md`
  (API freeze, benchmarks, optional-module extraction decisions).

## Hotfix

For a urgent fix on an already-released minor:

```bash
git checkout -b release/vX.Y vX.Y.0
# cherry-pick the fix
git tag -a vX.Y.Z+1 -m "vX.Y.Z+1"
git push origin release/vX.Y --tags
```

The `examples` job in CI re-runs against the tag automatically.
