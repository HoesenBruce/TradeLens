# Controlled publication evidence: v0.2.1

Date: 2026-10-07. Source SHA: b78c5925778831e7d2d039a4d0139ff1ddfc6e31.
Release: https://github.com/HoesenBruce/TradeLens/releases/tag/v0.2.1
Run: https://github.com/HoesenBruce/TradeLens/actions/runs/37590176429

All metadata, API/Web CI and both publication jobs succeeded after owner approval.
The v0.2.0 tag remains unchanged; its earlier publication failed before registry
login/build/push. #247 fixed the missing tag refs in the post-approval checkout;
#248 synchronized source metadata to 0.2.1.

## Paired image index digests

- ghcr.io/hoesenbruce/tradelens-api@sha256:9bf8d718c6c6df96fc02433e82bcb5584dff836d020f800b38a12e01a225263e
- ghcr.io/hoesenbruce/tradelens-web@sha256:a3694f5fc79aac7291948d829cda845e8e286fe2ce9e2fe7eeed9fe946d35f72

Both digest artifacts record source SHA above and version 0.2.1.

## Verified

- Both images pulled successfully by digest using a newly empty Docker config,
  with no login or credential helper. This verifies anonymous registry access.
- Actual anonymous registry manifests are saved in api-manifest.json and
  web-manifest.json. Both include linux/amd64 and linux/arm64. unknown/unknown
  descriptors are attestations and do not count as runtime architectures.
- Native Docker engine: aarch64. Pulled arm64 image labels for BOTH images:
  source=https://github.com/HoesenBruce/TradeLens;
  revision=b78c5925778831e7d2d039a4d0139ff1ddfc6e31; version=0.2.1.
- Existing scripts/deployment-smoke.py ran against the published API index digest,
  not a local build. Disposable SQLite/PostgreSQL with synthetic data/provider:
  all four restart/persistence and backup/restore PASS lines in api-smoke.log.
  Includes stopped API backup, PostgreSQL pg_dump/restore, attachment restoration
  and restored startup checks. This is API evidence, not browser E2E/live vendor evidence.
- Docker publishing and Release Please remain disabled_manually.

## Registry acceptance completed

Both official GitHub package pages explicitly display **Public** (api-public.png,
web-public.png). Package API access still returns 403 without read:packages; that
API response was not treated as evidence of visibility or absence. No visibility
setting was changed.

registry-evidence.json records anonymous registry byte-hashed index digests for
0.2.1, 0.2, 0, latest and sha-b78c5925778831e7d2d039a4d0139ff1ddfc6e31.
Every API alias resolves to the API index above; every Web alias to the Web index.
It also records both platform child digests and all four platform config labels:
source, revision and version match this release. Fresh empty-config pulls passed
again (anonymous-api.log, anonymous-web.log); temporary configs were removed.
Reviewed publisher emits only the two official names, with no compatibility aliases.
Stable behavior has live registry evidence; prerelease/manual-unversioned and
old-release rejection have regression evidence from scripts/test_docker_release_metadata.py
(PASS), not additional live publications. No new version was published for testing.

## Published-image runtime and browser acceptance

Host: macOS arm64, Docker Desktop aarch64 engine. runtime.json records exact image
references, running state, container uname and successful nginx configuration tests.

- **Native arm64 PASS:** API and Web, SQLite and PostgreSQL stacks, pinned index
  digests. Existing native API smoke evidence covers all four persistence/restore
  checks described above (api-smoke.log).
- **Emulated amd64 PASS:** API and Web pinned amd64 child digests. Docker Desktop
  executed linux/amd64 on arm64; uname inside both containers is x86_64. The
  QEMU/Rosetta backend was not independently identified. This is not native amd64
  evidence. amd64-api-smoke.log passes SQLite/PostgreSQL restart and backup/restore
  smoke, including SBI, attachments, analytics/value, news and synthetic HTTP bars.
  PostgreSQL service in that run is native arm64; API is emulated amd64.
- **Web E2E PASS:** Codex built-in browser against real published Web/API images;
  fresh setup, synthetic owner creation, explicit logout/login, Home render,
  navigation to Trades and refresh/SPA fallback. Native SQLite and PostgreSQL
  rendered a synthetic QA234 round trip (+US$100); amd64 exercised fresh empty state.
  Screenshots: arm64-home.png, postgres-trades.png, amd64-trades.png.
- **HTTP/production assets PASS:** production HTML, loaded JS/CSS, setup/status
  same-origin proxy and SPA fallback (web-http.json); nginx starts/tests successfully.
  No fatal browser exception observed. One nonfatal ViewTransition AbortError
  occurred during rapid navigation; final render/navigation remained functional.
- SQLite Home account-value chart used the disposable synthetic HTTP provider;
  no external vendor or real financial/account data was used. PostgreSQL was
  made ready before starting API. Initial QA-only provider/readiness setup was
  corrected without application changes.

All disposable Compose containers, networks and data volumes were removed after
validation, along with temporary Docker configs and provider process. Browser test
tabs were closed. Published images/packages were retained.

## Remaining #234 acceptance

OWNER/NAS REQUIRED:
- NAS anonymous pull and Web/API deployment/startup.
- NAS restart persistence, full backup/restore and restored startup.
- NAS update procedure preserving existing volumes; feed result back to #98.

AFTER NAS:
- Separately reviewed default prebuilt Compose migration to official GHCR images.
- Relevant CI, final deployment/updating documentation and update validation.

#233 registry acceptance is complete. #234 remains open for the above work.
#235 is not started. Publishers remain disabled_manually. No visibility setting,
Compose default, production database, mobile or tm-sync release was changed.
