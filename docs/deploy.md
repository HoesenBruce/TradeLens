> [!NOTE]
> **Upstream reference:** the docs site sources in [`marketing/content/docs/`](../marketing/content/docs/) — this copy documents the TradeLens fork. Upstream documentation may differ.

# TradeLens deployment
#
# Supported shapes:
#   0. Fork / one-click web on your Vercel or Cloudflare — see fork-deploy.md
#   1. Docker all-in-one (default / self-host)
#   2. Static SPA + API elsewhere (CORS + Server URL)
#   3. Static SPA with edge rewrite to API (same-origin from the browser)

> **Current TradeLens status:** official TradeLens GHCR images are published. `make up` (SQLite) and `make up-postgres` (PostgreSQL) use them by default. Source-build fallback remains available through `make up-build` / `make up-postgres-build`. Docker publisher and Release Please remain `disabled_manually`; image availability does not imply continuous publishing is enabled.
>
> **Database boundary:** TradeLens and upstream TraderMemos migration histories have diverged. Never hand a database used by one product to the other, including PostgreSQL. Use separate projects and fresh databases for different products. Switching an existing TradeLens source deployment to official TradeLens images is a same-product deployment change: take a complete backup, then preserve the checkout, Compose project, configuration and volumes.

## 0. Deploy web to *your* Vercel / Cloudflare (fork-friendly)

**Start here if you forked the repo or want one-click onto your own account:**

→ **[fork-deploy.md](fork-deploy.md)** (Vercel + Cloudflare buttons, import-your-fork steps, CORS)

Quick links (SPA only on Vercel/Cloudflare/Netlify; separate Railway API setup). Private repository access and provider authorization are required; provider deployment is unverified. These links do not deploy the Next.js marketing site or a complete stack:

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens&root-directory=web&project-name=tradelens&repository-name=tradelens&env=VITE_API&envDescription=Optional%20API%20base%20URL%20(e.g.%20https%3A%2F%2Fapi.example.com%2Fapi%2Fv1).%20Leave%20empty%20to%20set%20Server%20at%20login.&envLink=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens%2Fblob%2Fmain%2Fdocs%2Ffork-deploy.md)
[![Deploy to Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens%2Ftree%2Fmain%2Fweb)
[![Deploy to Netlify](https://www.netlify.com/img/deploy/button.svg)](https://app.netlify.com/start/deploy?repository=https://github.com/HoesenBruce/TradeLens)
[Railway API setup](https://github.com/HoesenBruce/TradeLens/blob/main/docs/fork-deploy.md#railway-api)

```bash
# On your API host after the UI is live:
TM_CORS_ORIGINS=https://*.vercel.app,https://*.pages.dev,https://*.workers.dev,https://*.netlify.app,https://*.up.railway.app,http://localhost:5173
```

---

## 1. Docker all-in-one (recommended default)

Run official TradeLens prebuilt images:

```bash
git clone https://github.com/HoesenBruce/TradeLens.git
cd TradeLens
cp .env.example .env
# Review TM_JWT_SECRET, set TM_ALLOW_INSECURE_JWT=false, and pin TM_IMAGE_TAG.
make up                 # SQLite
# OR, on a separate new installation:
make up-postgres        # PostgreSQL
# open http://localhost:3000
```

### Image registry and version

Compose loads `TM_IMAGE_REGISTRY` and `TM_IMAGE_TAG` from the root `.env`.
The registry defaults to `ghcr.io/hoesenbruce`, with image names `tradelens-api`
and `tradelens-web`. `DOCKERHUB_USERNAME` is no longer used; custom publishers
must set `TM_IMAGE_REGISTRY` and provide these TradeLens image names.
Production/self-hosted deployments should pin a stable version, such as
`TM_IMAGE_TAG=0.2.1` (the `.env.example` default). Without a tag setting,
Compose uses `latest`, a moving stable tag. For exact rollback/audit, use a
Compose override with each recorded `image@sha256:...` digest; record the paired
API/Web digests and matching database backup.

`make up-build` / `make up-postgres-build` build the current checkout as
`tradelens-api:local` / `tradelens-web:local`; registry overrides do not change
these local tags. `TM_IMAGE_TAG` supplies build metadata, not the source revision.

What you get:

| URL | Service |
|-----|---------|
| `http://localhost:3000` | nginx SPA |
| `http://localhost:3000/api/v1/*` | proxied → Go API |
| `http://localhost:8080` | API direct (optional; health, debug) |

Leave the login/settings **Server** field blank. The SPA uses relative `/api/v1`, and nginx proxies to the `api` container — no CORS required.

Important env (compose / host):

| Variable | Purpose |
|----------|---------|
| `TM_JWT_SECRET` | JWT signing secret — **required** for production (`openssl rand -hex 32`) |
| `TM_ALLOW_INSECURE_JWT` | Compose defaults `true` for first-run convenience; set `false`/unset in production |
| `TM_ALLOW_REGISTRATION` | Default `false`. After setup, only the owner exists unless you opt in |
| `TM_DATABASE_URL` | Unified DB URL — SQLite `sqlite:///data/tradermemos.db` (default) or `postgres://user:pass@host:5432/db?sslmode=require`. Legacy `TM_DB_PATH` still works for SQLite. |
| `TM_ATTACH_DIR` | Attachment disk path. Defaults to `<dbDir>/attachments` for SQLite; set explicitly for Postgres (e.g. `data/attachments`). |
| `TM_CORS_ORIGINS` | Leave empty for this mode |

**First boot:** open `http://localhost:3000` — if the database has no users, the **setup wizard** creates the owner (admin) account and an optional trading account. Public registration stays closed afterward.

```bash
# Production-ish compose example
cp .env.example .env
# edit .env: TM_JWT_SECRET=…, TM_ALLOW_INSECURE_JWT=false, TM_IMAGE_TAG=0.2.1
export TM_JWT_SECRET=$(openssl rand -hex 32)
export TM_ALLOW_INSECURE_JWT=false
make up
```

SQLite data and attachments live in `tm_data`. With the PostgreSQL overlay, the database lives in `tm_pg_data`, while attachments remain in `tm_data`. A complete instance backup requires **SQLite database + attachments**, or **PostgreSQL dump + attachments**, plus securely retained deployment configuration/secrets. Account ZIP exports and research Markdown exports are not complete instance backups. See the [backup/restore guide](../marketing/content/docs/self-hosting/backup-restore.mdx).

```bash
make logs        # follow compose logs
make down        # stop stack
```

Production tip: put **Caddy / Traefik / nginx** in front for **TLS** and point it at the `web` service only — `/api` stays same-origin. Do not expose the API without TLS on the public internet.

### Auth hardening (built-in)

- First-user **setup** endpoint; open `/auth/register` is disabled by default
- Passwords must be **≥ 10** characters (bcrypt)
- Auth + setup routes are **rate-limited** (~2 req/s per IP)
- Access vs refresh JWTs use distinct `typ` claims
- Server **refuses to start** on a known-insecure JWT secret unless `TM_ALLOW_INSECURE_JWT=true`

### API access tokens & OpenAPI docs

Create long-lived tokens in **Settings → API** for MCP, AI agents, or scripts. They have the same API power as your user account.

```bash
# Call any protected route with the token shown once at create time
curl -H "Authorization: Bearer tm_pat_…" https://your-host/api/v1/accounts
```

Browse the interactive OpenAPI reference (Scalar):

| Setup | Docs URL |
|-------|----------|
| Docker all-in-one (`web` nginx) | `http://localhost:3000/docs` |
| API directly | `http://localhost:8080/docs` |
| Split API host | `https://api.example.com/docs` |

Raw spec: `/openapi.yaml` on the same host. Docs are public; protect the host with TLS and network controls as usual.

---

## 2. Static web (Vercel / Cloudflare Pages / Netlify) + API elsewhere

Use when the UI is on a CDN and the journal API runs on a VPS, Fly, home NAS, etc.

```
https://app.example.com     → static SPA (CDN)
https://api.example.com     → Docker/Go API + SQLite volume
```

### API

1. Run the API container (or binary) with a reachable URL and persistent disk.
2. Allow the SPA origin:

```bash
TM_CORS_ORIGINS=https://*.vercel.app,https://*.pages.dev,http://localhost:5173
TM_JWT_SECRET=$(openssl rand -hex 32)
# Do not set TM_ALLOW_INSECURE_JWT on public APIs
# TM_ALLOW_REGISTRATION=true   # only if you want extra users via the UI
```

Wildcard forms `https://*.vercel.app` and `https://*.pages.dev` match preview/production CDN hosts. Use exact origins for custom domains.

### Web

Build the SPA and deploy `web/dist`:

```bash
cd web && vp install && vp build
```

Point the SPA at the API:

| Method | When |
|--------|------|
| Login / Settings → **Server** / **API server** | User brings their own API (runtime `tm_api_base`) |
| Build-time `VITE_API=https://api.example.com/api/v1` | Fixed public/demo API baked into the build |

Origin-only values (e.g. `https://api.example.com`) get `/api/v1` appended automatically.

Sample platform configs:

- [`web/vercel.json`](../web/vercel.json) — used by one-click / Git import (SPA)
- [`deploy/vercel.json.example`](../deploy/vercel.json.example) — optional `/api` edge rewrite (mode 3)
- [`deploy/cloudflare/_redirects.example`](../deploy/cloudflare/_redirects.example) — optional CF `/api` proxy

---

## 3. Static web + edge rewrite (same-origin CDN)

Keep the browser on one origin; the edge proxies `/api` to your API. No CORS and no Server field.

### Vercel

Copy [`deploy/vercel.json.example`](../deploy/vercel.json.example), set `destination` to your API host, deploy `web/dist` (or connect the `web/` project with `outputDirectory: dist`).

### Cloudflare Workers / Pages

Copy [`deploy/cloudflare/_redirects.example`](../deploy/cloudflare/_redirects.example) into `web/public/_redirects` before `vp build`, with a `200` proxy to your API. Keep SPA fallback in [`web/wrangler.toml`](../web/wrangler.toml) (`not_found_handling = "single-page-application"`) — do not add `/* /index.html 200` (Workers rejects it as an infinite loop).

Leave `TM_CORS_ORIGINS` empty when using rewrites — the browser never talks cross-origin.

---

## Choosing a mode

| Goal | Mode |
|------|------|
| Fork / one-click UI on **your** Vercel or CF | **[fork-deploy.md](fork-deploy.md)** |
| Homelab / VPS / NAS, one URL | **1. Docker** |
| Marketing/demo UI on CDN, users self-host API | **2. CDN + CORS** |
| Global SPA CDN, your hosted API, blank Server field | **3. Edge rewrite** |

Do **not** run the Go + SQLite API on Vercel serverless or Cloudflare Workers for v1 — keep the API on a machine/volume with a real disk.

---

## Checklist

- [ ] Changed `TM_JWT_SECRET` from the default
- [ ] SQLite/attachments on a persistent volume
- [ ] Docker: open the **web** port; Server field blank
- [ ] Split host: `TM_CORS_ORIGINS` matches the SPA origin(s)
- [ ] Uploads: nginx/proxy `client_max_body_size` ≥ API `TM_*_MAX_BYTES` (compose web image uses 20m)
