<div align="center">

<img src="brand/tradelens/icon.svg" width="88" alt="TradeLens" />

# TradeLens

### Record. Review. Improve.

**Self-hosted trading journal, portfolio analytics, and market review platform.** TradeLens is a Web-first fork of [TraderMemos](https://github.com/sinhong2011/TraderMemos). The existing Expo mobile client is preserved but is outside active fork validation.

TradeLens retains the original project’s AGPL-3.0 license and attribution; see [NOTICE](NOTICE) and [fork development policy](docs/FORK_DEVELOPMENT.md).

<br/>

[![Release](https://img.shields.io/github/v/release/HoesenBruce/TraderMemos-Private?color=8b5cf6&label=release)](https://github.com/HoesenBruce/TraderMemos-Private/releases) [![Web CI](https://github.com/HoesenBruce/TraderMemos-Private/actions/workflows/web-ci.yml/badge.svg)](https://github.com/HoesenBruce/TraderMemos-Private/actions/workflows/web-ci.yml) [![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue)](LICENSE)

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](api/go.mod) [![React](https://img.shields.io/badge/React-Vite+-61DAFB?logo=react&logoColor=black)](web/) [![SQLite](https://img.shields.io/badge/SQLite-embedded-003B57?logo=sqlite&logoColor=white)](api/) [![Self-hosted](https://img.shields.io/badge/Self--hosted-ready-8b5cf6)](docs/fork-deploy.md)

<br/>

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTraderMemos-Private&root-directory=web&project-name=tradermemos&repository-name=tradermemos&env=VITE_API&envDescription=Optional%20API%20base%20URL%20(e.g.%20https%3A%2F%2Fapi.example.com%2Fapi%2Fv1).%20Leave%20empty%20to%20set%20Server%20at%20login.&envLink=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTraderMemos-Private%2Fblob%2Fmain%2Fdocs%2Ffork-deploy.md) [![Deploy to Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTraderMemos-Private%2Ftree%2Fmain%2Fweb) [![Deploy to Netlify](https://www.netlify.com/img/deploy/button.svg)](https://app.netlify.com/start/deploy?repository=https://github.com/HoesenBruce/TraderMemos-Private) [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/new/template?template=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTraderMemos-Private&utm_medium=integration&utm_source=button&utm_campaign=tradermemos)

<br/>

[Quick start](#quick-start) · [Mobile app](#mobile-app) · [Upstream docs](https://trader-memos.vercel.app) · [Fork guide](docs/fork-deploy.md) · [Contributing](CONTRIBUTING.md) · [Design](DESIGN.md) · [License](LICENSE)

<br/>

<img src="docs/screenshots/tradelens-login-en.png" alt="TradeLens sign-in screen with the lens icon and product name" width="100%" />

</div>

## A look inside

<img src="docs/screenshots/tradelens-reports-en.png" alt="TradeLens English reports with equity curve and daily P&L" width="100%" />

<table>
<tr>
<td width="50%"><img src="docs/screenshots/tradelens-trades-en.png" alt="TradeLens English trade list" /><p align="center"><strong>Trade log</strong></p></td>
<td width="50%"><img src="docs/screenshots/tradelens-playbook-en.png" alt="TradeLens English playbook with per-setup performance" /><p align="center"><strong>Playbook</strong></p></td>
</tr>
</table>

Screenshots show the English TradeLens Web interface on a local instance. Trading results are generated demo data from `scripts/seed-demo.py`, not real trading performance.
Historical TraderMemos screenshots remain in `docs/screenshots/` for reference.

---

## Why TradeLens?

TradeLens keeps your trading journal and portfolio analytics on your own infrastructure. You control the database, configure your own OpenAI-compatible API for optional AI features, and can extend the application from source.

## What TradeLens adds

- **Portfolio review across currencies:** aggregate account statistics in an explicit target currency, with historical account-value estimates and visible warnings when valuation data is incomplete.
- **Japanese brokerage workflows:** SBI Securities execution and cash-statement imports, realized P&L report enrichment, and handling for margin-to-cash conversions, delivery settlements, and stock splits.
- **News and thesis review:** record news with linked assets and predictions, validate predictions against available market data, review outcomes, and export records as Markdown.
- **Localized Web review:** Simplified Chinese support and broader translation coverage across trading, import, analytics, and settings screens.

## Features

TradeLens inherits most core capabilities from TraderMemos while adding fork-specific portfolio, broker-import, and market-review features. Inherited upstream features may not all receive the same level of validation in this fork.

| Feature | What it does |
|---|---|
| 📊&nbsp; **Home** | Equity curve, expectancy, streaks, hold times, and annual goal pacing — per account or across a multi-account portfolio |
| 📒&nbsp; **Trade log** | Fills grouped into trades — filter, sort, tag, and drill into execution detail with MAE/MFE excursion charts |
| 🗓&nbsp; **P&L calendar** | Daily heatmap with weekly totals and day-detail drill-down |
| 📈&nbsp; **Reports** | Expectancy, SQN, Kelly %, MAE/MFE, Monte Carlo, execution quality — by setup, hour, and session, saved as view presets |
| 📖&nbsp; **Playbook** | Strategy library linked to the trades that used each setup, with rule-compliance scoring |
| 📥&nbsp; **Import** | Broker CSV presets (including SBI Securities), MT4/MT5 statements, IBKR Flex sync — or the [tm-sync](docs/tm-sync.md) watcher that imports statements as they appear |
| 🔔&nbsp; **Alerts** | Risk rules, daily loss limits, and prop-drawdown warnings — push and webhook, from your own server |
| ⏪&nbsp; **Bar replay** | Backtest symbols with available market data bar by bar against a persistent paper account — analyzed by the same reports |
| 🔗&nbsp; **Sharing** | Revocable read-only performance links, share cards, and a Year Wrapped recap |
| 🧮&nbsp; **Tools** | Position-size / FX / Kelly calculators, advanced chart, economic calendar, cash ledger |
| 🤖&nbsp; **AI** *(optional)* | Screenshot fill extraction + trade coach via OpenAI-compatible APIs — your keys |
| 🔌&nbsp; **API access** | Personal access tokens (`tm_pat_…`) for MCP/scripts; OpenAPI docs at `/docs` |
| 📱&nbsp; **Upstream mobile app** *(beta; unverified in this fork)* | iOS & Android companion (Expo) — offline journaling on both; widgets, Live Activities, and Siri / Action Button capture on iOS. TestFlight (iOS) · APK on [Releases](https://github.com/sinhong2011/TraderMemos/releases) (Android) |
| 🌗&nbsp; **Themes** | Dark and light, built on shadcn/ui + [coss ui](https://coss.com/ui/docs) tokens |

## Tech stack

| Layer | Stack |
|-------|-------|
| **API** | Go · Echo · sqlc · golang-migrate · SQLite / Postgres |
| **Web** | React · Vite+ · TanStack Router/Query/Form · Tailwind |
| **Mobile** | Expo (iOS & Android) · PanelUI + Uniwind (Tailwind) · iOS extras: WidgetKit / Live Activities / App Intents |
| **Sync agent** | [tm-sync](docs/tm-sync.md) — a Go binary that watches statement folders and imports on change |
| **Design** | shadcn/ui + coss ui tokens — see [DESIGN.md](DESIGN.md) |

## Quick start

Web UI and API in one Compose stack, on your own machine:

```bash
git clone git@github.com:HoesenBruce/TraderMemos-Private.git
cd TraderMemos-Private
make up-build    # → http://localhost:3000
```

Build from this fork so the Web UI includes TradeLens branding; `make up` pulls upstream images by default until fork release metadata is updated (#103). Same-origin `/api` — no CORS, leave the **Server** field blank. On first visit the **setup wizard** creates your owner account.

<details>
<summary><strong>Options and production notes</strong></summary>

<br/>

```bash
cp .env.example .env   # optional: DOCKERHUB_USERNAME, TM_IMAGE_TAG
make up                # pulls upstream TraderMemos images by default
make up-build          # or build both images from this repo
```

For production, set `TM_JWT_SECRET=$(openssl rand -hex 32)` and put TLS (Caddy/Traefik) in front — see [docs/deploy.md](docs/deploy.md).

Prefer Postgres over the default SQLite? `make up-postgres-build` builds and runs the same stack with a Postgres overlay.

</details>

## Deploy to the cloud

Prefer a hosted SPA with the API elsewhere? The buttons at the top split the app across two hosts:

| Button | Deploys |
|--------|---------|
| Vercel / Cloudflare / Netlify | Web SPA (`web/`) |
| Railway | Go API ([`railway.toml`](railway.toml) → `api/Dockerfile`) — attach a Volume at `/data` |

**1. Deploy the web UI** — click a button above; it hosts the SPA on **your** account.

**2. Run the API** — use the Railway button, or point it at any host running `api/Dockerfile`.

**3. Connect them** — allow your CDN origin on the API:

```bash
TM_CORS_ORIGINS=https://*.vercel.app,https://*.pages.dev,https://*.workers.dev,https://*.netlify.app,https://*.up.railway.app,http://localhost:5173
```

Then open the web app and set **Server** to your API URL (or bake it in with `VITE_API` at build time).

Already forked? Import your fork → Root **`web`** (Vercel/CF) or [`netlify.toml`](netlify.toml) / [`railway.toml`](railway.toml). Details: [docs/fork-deploy.md](docs/fork-deploy.md).

## Mobile app

The preserved upstream companion app (beta) is not validated for this fork. It journals against your own server — enter your instance URL at login, same as the web app's **Server** field.

- **iOS** — distributed through TestFlight while in beta.
- **Android** — download `TraderMemos-<version>.apk` from the [latest release](https://github.com/sinhong2011/TraderMemos/releases/latest) and sideload it (allow *Install unknown apps* for your browser or file manager; a `.sha256` file ships next to the APK for verification). There is no Play Store listing.

Building it yourself instead: see [mobile/README.md](mobile/README.md).

## Development

```bash
git clone git@github.com:HoesenBruce/TraderMemos-Private.git
cd TraderMemos-Private
make setup && make dev   # API :8080 + web :5173
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for targets, Vite+ commands, and project structure.

### Demo data

Fill any instance with ~200 generated trades over the last three months:

```bash
make demo-seed API=http://localhost:3000/api/v1 EMAIL=you@example.com PASSWORD='…'
```

It talks to the public API only, so it works against a local stack or a deployed demo. Re-running is a no-op — fills are deduplicated server-side.

For the preserved upstream demo book — 172 trades across 13 months, with
notes, setups and mistake tags — import [`docs/demo/tradermemos-demo-trades.json`](docs/demo/)
instead, either from **Import** in the web app or with the CLI:

```bash
tradermemos import --account <account-id> --file docs/demo/tradermemos-demo-trades.json
```

## Docs

Upstream user docs (which may differ from this fork) live on the docs site: **[trader-memos.vercel.app](https://trader-memos.vercel.app)** — getting started, importing trades, supported brokers, self-hosting guides, FAQ, and the full API reference.

| Doc | Topic |
|-----|--------|
| [docs/features/analytics-metrics.md](docs/features/analytics-metrics.md) | Analytics formulas, sample rules, units, and edge cases |
| [docs/fork-deploy.md](docs/fork-deploy.md) | One-click / fork → Vercel, Cloudflare, Netlify, Railway |
| [docs/deploy.md](docs/deploy.md) | Docker, CORS, edge rewrite |
| [docs/tm-sync.md](docs/tm-sync.md) | The tm-sync local statement watcher |
| [docs/release.md](docs/release.md) | Versioning, changelogs, GitHub Releases |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Local dev (`make dev`) |
| [DESIGN.md](DESIGN.md) | UI system — shadcn/ui + coss ui tokens |

## Upstream live demo

Try the original TraderMemos (not a TradeLens deployment) without installing anything: **[tradermemos.netlify.app](https://tradermemos.netlify.app)**

Sign in with `tradermemosdemo` / `demopassword`. The demo account carries a seeded dataset — treat it as a shared sandbox, and don't store anything real in it.

If TraderMemos is part of your daily review, consider [sponsoring its development](https://github.com/sponsors/sinhong2011) — it keeps the project free and self-hosted for everyone.

## Upstream star history

<div align="center">
<a href="https://www.star-history.com/#sinhong2011/TraderMemos&Date">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=sinhong2011/TraderMemos&type=Date&theme=dark" />
    <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=sinhong2011/TraderMemos&type=Date" />
    <img alt="Star history chart for TraderMemos" src="https://api.star-history.com/svg?repos=sinhong2011/TraderMemos&type=Date" width="600" />
  </picture>
</a>
</div>

## License

TradeLens is based on [TraderMemos](https://github.com/sinhong2011/TraderMemos), originally developed by [sinhong2011](https://github.com/sinhong2011) and contributors.

TradeLens is distributed under the GNU AGPL-3.0. See [LICENSE](LICENSE) for the full license terms and [NOTICE](NOTICE) for attribution.
