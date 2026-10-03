# Contributing to TradeLens

## Git workflow

This repository uses a **main-based, short-lived branch workflow**.

`main` is the stable integration branch and must remain runnable. Do not perform feature
development directly on `main`.

### Branch naming

| Branch | Purpose |
|---|---|
| `main` | Stable integration branch; must remain runnable |
| `feat/*` | New features |
| `fix/*` | Bug fixes |
| `refactor/*` | Refactoring without intended behavior changes |
| `docs/*` | Documentation-only changes |
| `chore/*` | Tooling, CI, dependencies, build, and maintenance work |

Prefer issue-linked branch names when an issue exists:

```text
feat/116-jp-broker-accounting
fix/103-stock-name-display
refactor/140-broker-accounting-engine
docs/150-development-workflow
chore/160-update-dependencies
```

### Standard development flow

1. Start from the latest `main`.
2. Create a short-lived branch for one coherent issue or change.
3. Implement and validate the change on that branch.
4. Open a pull request targeting `main`.
5. Complete the required tests and review.
6. Merge only when `main` will remain runnable.
7. Delete the merged branch when it is no longer needed.

Typical flow:

```text
Issue
  ↓
main
  ↓
feat/* / fix/* / refactor/* / docs/* / chore/*
  ↓
implementation
  ↓
tests
  ↓
PR → main
  ↓
merge
  ↓
delete branch
```

### Main branch rules

- Do not develop features directly on `main`.
- Do not use `main` as a scratch or experiment branch.
- `main` must remain runnable and should keep relevant tests passing.
- New branches should normally start from the latest `main`.
- Release, tag, and deployment work should normally be based on `main`.
- Do not introduce a long-lived `develop` branch unless this policy is explicitly changed.

### Issue, branch, and PR scope

The default relationship is:

```text
1 Issue ≈ 1 Branch ≈ 1 Pull Request
```

This is a default, not an absolute rule. A pull request should still represent one coherent
change even when multiple tightly coupled issues are involved.

When a bug is found while a pull request is still open:

- fix it in the same branch when it is directly caused by or required for that change;
- create a separate issue and `fix/*` branch when it is unrelated or already exists on `main`.

Avoid combining unrelated feature work, bug fixes, refactors, and maintenance in one pull
request.

### Worktrees and parallel development

When multiple issues are developed concurrently, prefer separate Git worktrees instead of
repeatedly switching branches in one working directory.

Recommended model:

```text
TradeLens/                   main
../tm-116/                    feat/116-jp-broker-accounting
../tm-118/                    feat/118-genbiki
../tm-i18n/                   feat/120-i18n
```

For AI-assisted development, prefer:

```text
1 Worktree = 1 Branch = 1 Issue = 1 Codex/agent workspace
```

This reduces context leakage, unrelated file changes, and accidental cross-issue commits.

### Keeping a branch up to date

For short-lived private branches, prefer rebasing onto the latest `main` before final review
when practical:

```bash
git fetch origin
git rebase origin/main
```

Do not rewrite shared branch history casually. If a branch is being used by multiple people,
coordinate before rebasing or force-pushing.

### Upstream synchronization

This fork should keep the remote roles clear:

```text
origin   → this fork
upstream → original TraderMemos repository
```

Integrate upstream changes into this fork's `main` first, verify compatibility, and create new
feature/fix branches from the updated `main`.

Preferred direction:

```text
upstream/main
      ↓
fork main
      ↓
feat/* / fix/* / refactor/* / docs/* / chore/*
```

Do not routinely branch new fork-specific work directly from `upstream/main`.

## Development setup

### Prerequisites

- **Go** 1.27+ (via [mise](https://mise.jdx.dev/) — see `mise.toml`)
- **Node** 24 LTS (via Vite+ `web/.node-version` or mise) + **pnpm** 11 + **Vite+** (`vp` CLI)
- **sqlc** (optional; for regenerating store code)

### Clone and bootstrap

```bash
git clone git@github.com:HoesenBruce/TradeLens.git
cd TradeLens
make setup           # mise + air + vp install; seeds api/.env

# Web validation (from repo root)
make check           # go vet + vp check
make test            # go test + vp test

# Optional: edit api/.env (TM_JWT_SECRET, TM_DATABASE_URL, …)
# See api/.env.example for the full list.

make dev             # API :8080 (air) + web :5173 (vite) with hot reload
```

SQLite under `api/data/` is the zero-config path — no Postgres/Redis required.

Useful targets:

| Target        | What it does                          |
|---------------|----------------------------------------|
| `make dev`    | API + web together (Ctrl+C stops both) |
| `make dev-api`| API only (air)                         |
| `make dev-web`| Vite+ dev server only                   |
| `make kill`   | Free ports 8080 / 5173 + air processes |
| `make check`  | Go vet + Vite+ check (lint/fmt/types) |
| `make test`   | Go + web unit tests                    |
| `make sqlc`   | Regenerate `api/internal/store`        |

Vite+ proxies `/api` → `http://localhost:8080` during `vp dev`.

### Self-host / deploy

See **[docs/fork-deploy.md](docs/fork-deploy.md)** to put the SPA on your Vercel/Cloudflare account, and **[docs/deploy.md](docs/deploy.md)** for Docker / CORS / edge rewrite. Deploy buttons: [README](README.md).

```bash
make up          # pull Hub images: web :3000 (SPA + /api proxy), api :8080
make up-build    # build Dockerfiles from this checkout instead
make down
make logs
```

Hub namespace / tag: copy [`.env.example`](.env.example) → `.env` and set `DOCKERHUB_USERNAME` / `TM_IMAGE_TAG`. CI publish uses GitHub secrets `DOCKERHUB_USERNAME` + `DOCKERHUB_TOKEN`.

### Vite+ commands (run from `web/`)

| Command | What it does |
|---------|--------------|
| `vp dev` | Dev server (:5173) |
| `vp build` | Production bundle |
| `vp test` | Unit tests (Vitest) |
| `vp check` | Lint + format + typecheck |
| `vp fmt` | Format only |
| `vp staged` | Check staged files (also runs on pre-commit) |
| `pnpm run …` | Same scripts via pnpm (`dev`, `test`, `build`, …) |

### Project structure

```
api/         Go backend (Echo, sqlc, golang-migrate, SQLite)
web/         React SPA (Vite+, TanStack Router)
mobile/      Expo app (iOS & Android)
docs/        Specs / roadmaps
DESIGN.md    Signal Terminal design system — read before UI work
```

### Design

UI work must follow `DESIGN.md` (Signal Terminal). Do not invent alternate type/color/radius without explicit approval.

## Optional ReUI resources

ReUI is an optional external development resource. TradeLens retains licensed
MIT components and examples, but does not vendor ReUI skills or their documentation.
Contributors can obtain those materials through the
[official ReUI documentation and installer](https://reui.io/docs/agent-skills).
