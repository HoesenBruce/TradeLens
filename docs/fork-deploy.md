> [!NOTE]
> **Upstream reference:** the docs site sources in [`marketing/content/docs/`](../marketing/content/docs/) — this copy documents the TradeLens fork. Upstream documentation may differ.

# Fork → deploy the web UI on your account

> TradeLens is based on [TraderMemos](https://github.com/sinhong2011/TraderMemos). The current fork is `HoesenBruce/TradeLens`; private-repository access is required. Stable deployment identifiers remain compatible; see [branding compatibility](#branding-compatibility).

Goal: another GitHub user gets **TradeLens web** on **their** Vercel / Cloudflare / Netlify, and/or the **API** on Railway, then connects them.

```
You                    CDN (Vercel / CF / Netlify)     API (Railway / Docker)
├─ one-click web ───► SPA                     ──CORS──► Go + SQLite volume
└─ login “Server” = https://your-api.up.railway.app
```

---

## Path A — One-click (easiest)

No manual fork. The platform clones into *your* GitHub and deploys under *your* account.

| Platform | Button / link | What you get |
|----------|---------------|--------------|
| **Vercel** | [Deploy with Vercel](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens&root-directory=web&project-name=tradelens&repository-name=tradelens&env=VITE_API&envDescription=Optional%20API%20base%20URL%20(e.g.%20https%3A%2F%2Fapi.example.com%2Fapi%2Fv1).%20Leave%20empty%20to%20set%20Server%20at%20login.&envLink=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens%2Fblob%2Fmain%2Fdocs%2Ffork-deploy.md) | Full monorepo clone; project Root = `web` |
| **Cloudflare** | [Deploy to Cloudflare](https://deploy.workers.cloudflare.com/?url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens%2Ftree%2Fmain%2Fweb) | New repo from `web/` only; Workers static SPA |
| **Netlify** | [Deploy to Netlify](https://app.netlify.com/start/deploy?repository=https://github.com/HoesenBruce/TradeLens) | Uses root [`netlify.toml`](../netlify.toml) (`base = web`) |
| **Railway** | [Deploy on Railway](https://railway.com/new/template?template=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens&utm_medium=integration&utm_source=button&utm_campaign=tradelens) | Go API via [`railway.toml`](../railway.toml); attach Volume at `/data` |

Optional env **`VITE_API`**: bake in a default API base (`https://api.example.com/api/v1`). Leave blank to type the Server URL at login.

---

## Path B — You already forked

Use this when you clicked **Fork** on GitHub and want continuous deploys from *your* fork.

### Vercel

1. Open [vercel.com/new](https://vercel.com/new) → **Import** your fork (`youruser/TraderMemos`).
2. Set **Root Directory** to `web` (important).
3. Leave build settings alone — [`web/vercel.json`](../web/vercel.json) supplies install/build/output + SPA rewrites.
4. Optional: Environment Variable `VITE_API`.
5. Deploy. Later pushes to your default branch redeploy automatically.

### Cloudflare

**Option 1 — Workers (matches one-click config)**  
1. [Deploy to Cloudflare](https://deploy.workers.cloudflare.com/?url=https%3A%2F%2Fgithub.com%2FYOURUSER%2FTraderMemos%2Ftree%2Fmain%2Fweb) — replace `YOURUSER`, or connect the `web/` app in the dashboard.  
2. Uses [`web/wrangler.toml`](../web/wrangler.toml) (`assets` + SPA `not_found_handling`).

**Option 2 — Pages Connect to Git**  
| Setting | Value |
|---------|--------|
| Repository | your fork |
| Root directory | `web` |
| Build command | `pnpm run build` |
| Output directory | `dist` |
| Env (optional) | `VITE_API` |

Prefer Option 1 (Workers + [`web/wrangler.toml`](../web/wrangler.toml)) for SPA routing — Cloudflare rejects a `/* /index.html 200` `_redirects` rule on Workers assets.

### Netlify

1. [app.netlify.com/start/deploy](https://app.netlify.com/start/deploy?repository=https://github.com/HoesenBruce/TradeLens) — or **Add new site → Import** your fork.
2. Root [`netlify.toml`](../netlify.toml) already sets `base = web`, build, publish, and SPA redirect.  
3. Optional env: `VITE_API`.  
4. Allow `https://*.netlify.app` in `TM_CORS_ORIGINS`.

### Railway (API)

Railway is the best one-click host for the **Go API** (disk volume for SQLite). Pair it with a CDN web deploy above.

1. [Deploy on Railway](https://railway.com/new/template?template=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens&utm_medium=integration&utm_source=button&utm_campaign=tradelens) — or New Project → Deploy from GitHub → your fork.
2. Root [`railway.toml`](../railway.toml) builds `api/Dockerfile` and health-checks `/healthz`.  
3. **Attach a Volume** mounted at `/data` (keeps SQLite + attachments across deploys).  
4. Variables:
   | Variable | Notes |
   |----------|--------|
   | `TM_JWT_SECRET` | Required — generate with `openssl rand -hex 32` |
   | `TM_CORS_ORIGINS` | e.g. `https://*.vercel.app,https://*.netlify.app` |
   | `TM_DATABASE_URL` | Default `sqlite:///data/tradermemos.db` (matches Dockerfile); or `postgres://user:pass@host:5432/db?sslmode=require`. Legacy `TM_DB_PATH` still works for SQLite. |
5. Generate a public domain (`*.up.railway.app`) → use that as login **Server** / `VITE_API`.  
6. `PORT` is honored automatically when `TM_HTTP_PORT` is unset.

---

## Path C — Point the UI at your API

The CDN only hosts the SPA. Your journal data stays on a host you control.

```bash
# On the API host (Docker / binary)
TM_JWT_SECRET=$(openssl rand -hex 32)
TM_CORS_ORIGINS=https://*.vercel.app,https://*.pages.dev,https://*.workers.dev,https://*.netlify.app,https://*.up.railway.app,http://localhost:5173
# add your custom domain exactly, e.g. https://journal.example.com
```

Then either:

- Leave `VITE_API` empty → open the site → **Server** = `https://api.your.domain` (or full `.../api/v1`), or  
- Set `VITE_API=https://api.your.domain/api/v1` on the CDN project and redeploy.

API Docker / compose: see [deploy.md](deploy.md).

---

## Checklist for fork deployers

- [ ] UI live on your Vercel, Cloudflare, or Netlify account  
- [ ] API running (Railway / Docker / VPS) with a public HTTPS URL and persistent SQLite volume  
- [ ] `TM_CORS_ORIGINS` includes your CDN host pattern (or exact custom domain)  
- [ ] Login works with **Server** set (or `VITE_API` baked in)  
- [ ] Changed `TM_JWT_SECRET` from the default  

---

## Why not put the API on Vercel/Workers?

The API is Go + SQLite + uploads. Keep it on Docker/VPS/NAS. The one-click buttons are **web-only** by design.

---

## Maintainer tip (upstream)

In GitHub → **Settings → General → Template repository**, enable the template flag so “Use this template” appears next to Fork. One-click Deploy buttons already clone without requiring a template.

## Branding compatibility

> **Current TradeLens status:** official TradeLens images are not yet published. Docker image publishing and Release Please are disabled. Use `make up-build` (SQLite) or `make up-postgres-build` (PostgreSQL) to build this checkout.
>
> **Database boundary:** `make up` and `make up-postgres` currently pull upstream TraderMemos images (`sinhong2011/tradermemos-*`) by default. Upstream and TradeLens migration histories have diverged. Never alternate them against the same existing database volume, including PostgreSQL. Use a separate Compose project and fresh database/volumes for a different product; retain a complete backup before any migration.

Source builds use `tradelens-api` / `tradelens-web` image names and TradeLens OCI
labels. The disabled publish workflow targets only GHCR `hoesenbruce/tradelens-api`
and `hoesenbruce/tradelens-web`, without legacy aliases. Official images remain
unavailable until controlled first publication; see [release policy](release.md).
Default upstream pulls remain unchanged; use `make up-build` for this fork.
No volume or Compose project rename is required.

Keep existing Compose service keys, `tm_data`, database filenames, `TM_*` settings,
CLI paths and cloud deployment identifiers: renaming them can disconnect stored
data or create a second deployment. Existing cloud projects need no migration;
new Vercel projects default to `tradelens`.

New Android release artifacts use `TradeLens-<version>.apk` and `TradeLens.apk`;
`TraderMemos.apk` and its checksum remain aliases for existing download links.
Historical releases are unchanged. Mobile app identity/display configuration is
retained pending mobile reactivation; iOS/Android builds are not validated here.
PWA metadata already uses TradeLens. Account export download names now use
`tradelens-export`; the account export format and import compatibility are unchanged. Account ZIP
exports and research Markdown exports are not full-instance backups; retain the
database (or PostgreSQL dump) together with attachments.

## Repository rename (issue #104)

The private repository is now `HoesenBruce/TradeLens`. Existing clones can keep
their directory names; update the remote in each development/NAS checkout:

```sh
git remote set-url origin git@github.com:HoesenBruce/TradeLens.git
git remote -v
git fetch origin
git ls-remote upstream HEAD
```

Keep `upstream` at `https://github.com/sinhong2011/TraderMemos.git`. The optional
`fork` remote (`HoesenBruce/TraderMemos`) is a separate repository, not the renamed
private repository. GitHub resolves the old private repository URL to TradeLens;
update bookmarks, scripts and deployment provider Git integrations to the new URL
instead of relying indefinitely on that redirect.

For an existing NAS install, keep its checkout directory, Compose project name,
`.env`, bind mounts and named volumes. After updating origin, use the existing
source-build update procedure (`git pull --ff-only`, then `make up-build`). Do not
clone into a new directory or run `docker compose down -v`. Docker Hub image
names and namespaces are independent of the GitHub rename; this fork does not
publish GHCR images. Existing image-based installs keep their configured tags.

Actions use relative reusable workflows and `GITHUB_REPOSITORY` for release
uploads, so no hard-coded private repository name needs changing. Check external
PAT repository selections, deployment integrations and Docker environment
approvers on each existing environment. Live NAS/cloud deployment and a real
release require access to those environments and are not established by a local
source check.
