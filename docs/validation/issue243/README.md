# #243 local validation — 2026-10-08

Status: implementation validated locally; **Render deployment and external HTTPS
acceptance are pending**. No live URL is asserted and README is unchanged.

## Source assessment before implementation

- Started from `origin/main` `ba4b0c18` on `feat/243-render-demo`, clean checkout.
- Official v0.2.1 is published; annotated tag `98e47f37` resolves to source
  `b78c5925778831e7d2d039a4d0139ff1ddfc6e31`. `docs/release.md` and the existing
  #234 validation record establish the split API/Web GHCR publication. #243's
  conditional assumption that images might not yet exist is obsolete.
- Current main includes the fixed fictional JPY SBI/news showcase added after
  v0.2.1. The demo therefore builds current source, rather than trying to turn
  the existing two-image release into a single-service runtime.
- Inspected both Dockerfiles and SQLite/build/Postgres Compose configurations.
  They and all GHCR/release workflows remain unchanged by this implementation.
- `seed-demo.py` legacy generation is not used. `--mode showcase` delegates to
  `seed-showcase.py`, which refuses nonempty users rather than promising a safe
  rerun. Startup creates a fresh private SQLite DB on every boot and reuses this
  exact seed path plus the loopback-only `showcase-market.py` provider.
- Checked current Render documentation for Free sleeping/ephemeral files,
  single public port, HTTP health checks and Blueprint fields. Links are in the
  deployment guide. No Render account, paid service or public instance was created.

## Checks passed

- `cd api && go test ./...`: all packages passed; [output](go-test.txt).
- `cd api && go vet ./...`: exit 0, no diagnostics.
- Focused demo policy tests, including normal self-hosted configuration:
  [output](demo-go-test.txt).
- `python3 scripts/test_showcase.py`: deterministic fixture checks passed.
- `docker build -f deploy/demo/Dockerfile -t tradelens-demo:243 .`: complete API
  and React production build, native Linux ARM64 image on this machine.
- `python3 scripts/test_demo_render.py`: [output](container-test.txt). Runtime
  constrained to 512 MiB / 0.5 CPU; readiness timings recorded in the output over two boots;
  about 32 MiB memory observed after checks (not a load-test or peak-memory claim).
- Missing JWT fails startup; inherited production DB URL/integration flags are
  ignored; fresh boot and restart produce one user/account, seven closed trades,
  three news entries and three predictions without duplicates. Reset changes
  user identity and rejects old tokens. Public user is not an admin.
- Password login, invalid credentials, token refresh and unauthenticated read
  rejection passed. Backend writes, registration/setup, credentials/TOTP, API
  tokens, uploads, admin/configuration, broker sync, AI and sharing are rejected.
  Direct loopback API write is also denied, bypassing nginx.
- SBI portfolio points are all complete: six days, final value JPY 975,000,
  contributed capital JPY 950,000, realized P&L JPY 25,000. News performance is
  six horizons: four pending, two unavailable. Prediction histories are readable.
- Provider-process death exits the entire service with failure; clean shutdown
  and restart work. Root redirect stays relative behind a mapped HTTP port and
  the demo entry is served as HTML.

## Built-in browser acceptance

Used Codex built-in browser against the real container API on localhost:19000,
after readiness succeeded, with a fresh seeded database. No mock API or real
brokerage data was used.

- Root → demo notice → login; invalid password rejected, correct public password
  accepted; no public registration affordance.
- Previous token after replacement redirected to login with session-expired
  notice. Logout returned to login and re-login succeeded.
- JPY totals and Tokyo dates loaded; both Home charts rendered fixture data with
  a range covering September 1–8 (90 days and All exercised). Initial 30-day
  empty state was also observed. The entry guide explicitly recommends All so
  the fixed fixture remains discoverable as the current date advances.
- News list/detail, predictions and performance report rendered with expected
  status counts. Editing a title then cancelling preserved the original title
  when reopened. Saving was rejected with the demo read-only message; closing
  and re-entering showed the original data.
- SBI 6758 full detail displayed 信用转现物 with original 現引 audit identity.
- Public user profile displayed member role. Restricted settings/history can
  show normal unavailable/error states; editing controls remain visible but the
  backend rejects submission, as documented.

Screenshots: [entry](entry.jpg), [valuation charts](home.jpg),
[news performance](news.jpg), [SBI detail](sbi.jpg),
[backend read-only refusal](read-only.jpg).

## Remaining external acceptance

Render Dashboard owner must create the Free service from the reviewed branch,
generate JWT secret, confirm no paid resources/disk, and test the assigned HTTPS
URL, Render readiness/redeploy and actual 15-minute sleep/wake cycle using
[`docs/demo-render.md`](../../demo-render.md). Render's Linux AMD64 build and
runtime, public TLS/proxy and workspace quota behavior are not established by
this local ARM64 run. The final public URL must be verified before a README link.

Mobile was not validated; outside current fork scope. No live integrations,
real-market data, load testing, production migration, release publication or
self-hosted persistence changes were required or performed.
