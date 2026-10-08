# Public fictional demo on Render Free

This is a separate, read-only product-evaluation deployment for #243. It uses
one Docker web service: nginx serves the React SPA and proxies the Go API on
the same public port. Python supervises startup and the existing fictional
market-data fixture. No database service, disk, paid worker, external AI key,
broker connection, GHCR credential, or real user data is required.

## Deploy (manual Dashboard actions)

1. Connect `HoesenBruce/TradeLens` in Render and create a Blueprint using
   [`render.yaml`](../render.yaml) from the reviewed commit/branch. For pre-merge
   acceptance explicitly choose `feat/243-render-demo`; after merge choose main.
   Alternatively create one Docker Web Service with repository-root context and
   Dockerfile `deploy/demo/Dockerfile`.
2. Confirm the instance is **Free**, one instance, with **no disk** and no other
   services. Confirm the workspace's free-hour and build/bandwidth allowances
   are available. Do not upgrade or add paid resources to make this demo work.
3. Set `TM_JWT_SECRET` to a Render-generated random secret of at least 32
   characters (`generateValue: true` in the Blueprint). It is a server secret;
   never use the public demo password as this value. No other variables are
   required. Leave `PORT` at Render's default 10000, health-check path `/readyz`.
4. Build uses the Dockerfile; start uses its entrypoint. Do not override the
   Docker command. Automatic deploys are off so fixture changes get reviewed.
5. Deploy and wait for readiness. Open the assigned HTTPS URL, which redirects
   to `/demo`, read the fiction/reset notice, and use **Try Demo** to log in:
   `demo@example.com` / `fictional-demo-password`. These are deliberately public
   evaluation credentials, never credentials for a real installation.
6. Perform the external acceptance checklist below and record the exact URL,
   commit and results in the PR. Only then consider a separate README Demo link.
   This implementation does not add a URL to README or assert a live deployment.

## Startup, reset and health

Every process start allocates a new `tradelens-demo-*` temporary directory and
SQLite database. The API migrates it, first-user setup runs privately, and
`scripts/seed-demo.py --mode showcase` dispatches to `seed-showcase.py`. The runtime freezes its startup date in Asia/Tokyo and maps the showcase
events across the preceding 30 days using the existing versioned JP calendar.
Trading days, cash flows, news and fictional prices share this one timeline;
the entry page displays the actual generated range. Warm requests reuse it.
The seed
imports the fictional SBI JPY history, journals, cash flows, news and predictions.
The bootstrap API is stopped, the user is demoted from admin, and the final API
starts with `TM_DEMO_MODE=true`. Only then does nginx bind `0.0.0.0:$PORT`.
The demo account's display and market timezone are seeded to Asia/Tokyo with
JPY display currency so SBI dates agree with the fixture.
No public traffic can reach the writable bootstrap server. Its separate random
JWT secret also prevents bootstrap tokens from authenticating to the final API.

Reset via **Manual Deploy → Deploy latest commit** or Render's restart control.
Every restart reseeds, even if files survive a particular restart. There is no
public reset endpoint and no caller-controlled delete path. Partial initialization
fails the container rather than opening an incomplete demo. Old login tokens
stop working after reset; reload and log in again. No migrations or recovery need
to be run manually in a shell.

`/healthz` checks the API process. `/readyz` checks the API, SQLite access and
fictional provider; it is the Render and Docker health check. The supervisor
terminates all children and exits if API, provider or nginx exits. Render can
restart the service. SIGTERM drains the API before child cleanup completes.

## Security boundary and supported workflows

The demo API has an explicit allowlist of read routes plus normal password login
and refresh. Authentication and ownership checks still run. All writes, unknown
routes, public registration/setup, user deletion/password/TOTP changes, API-token
management, CSV/file/media/OCR uploads, broker sync, share links, AI calls,
notifications and sensitive administration/configuration reads are rejected by
the backend with HTTP 403 `demo_read_only`. The UI may still show normal editing
controls; submitting them is rejected. Use a private self-hosted instance for
editing/import evaluation. Exports are also disabled in this first demo.

Demo config forcibly disables registration, insecure JWT mode, AI/OCR, sharing,
economic-feed access and scheduled jobs, and forces the loopback HTTP market
provider. The container sanitizes inherited `TM_*` settings before bootstrap,
so an accidental production DB URL/key is not used. Do not connect real storage
or put private data in this deployment. `TM_DEMO_MODE` is off by default elsewhere;
the normal Dockerfiles, Compose volumes and publishing workflows are unchanged.

The fixture listens only on `127.0.0.1:18964`; neither it nor the API/bootstrap
ports are published. The provider has no network-download fallback. Unknown
symbols/resolutions/dates fail or return no fixture data. In the Render demo, daily fictional JPY bars cover the Japanese sessions in
the **30 days before startup**, excluding the startup date. Six ordered showcase
event dates are distributed across that window; intervening sessions use the
previous fictional event price. These are explicitly generated practice prices,
not repaired or downloaded market data. Home’s default 30-day charts include
the sample; select JPY and **1D** for trade charts. The entry page shows the
current window. Dates remain fixed while the instance stays warm and move
forward on its next restart/cold start.

Standalone `seed-demo.py --mode showcase` / `showcase-market.py` retain the
original September 1–8, 2026 timeline unless the demo runtime opts in with its
shared `TRADELENS_SHOWCASE_TODAY` date. No self-hosted persistence behavior changes.
The existing JP calendar snapshot covers 2020–2030; startup fails outside its
coverage rather than guessing trading days, so extend that snapshot before
expiry. Seven closed trades produce JPY 25,000 realized P&L; closing
estimated account value is JPY 975,000 on contributed capital JPY 950,000.
News shows three fictional catalysts and six horizon outcomes (four pending,
two unavailable). These statuses are intentional fixture coverage, not market
recommendations. Previously generated validation history is readable; triggering
a new evaluation is a write and is disabled.

## Free-tier behavior

[Render Free](https://render.com/docs/free) sleeps after 15 minutes without inbound
traffic and can take about a minute to wake. Local files are lost on redeploy,
restart or sleep. This demo rebuilds its small fixture automatically; the public
listener opens only after initialization. Render's loading page can appear while
waking. There is no uptime/SLA guarantee; free-hour, build-minute and bandwidth
quotas can suspend availability. Do not add a keep-alive monitor to defeat sleep.
Free-tier quotas are shared with other services in the workspace; keep the
deployment at $0 by checking Dashboard usage and disabling/removing the service
when allowances run out. This design provides no paid fallback.

## Validation

Local reproducible checks:

```sh
cd api
go test ./...
go vet ./...
cd ..
docker build -f deploy/demo/Dockerfile -t tradelens-demo:243 .
python3 scripts/test_demo_render.py
```

The container test caps runtime at 512 MiB / 0.5 CPU and checks two clean boots,
duplicate-free seeding, stale-token rejection, a hostile inherited DB URL,
backend write rejection, authentication, SBI valuation and news read-back. This
is local resource-limit evidence, not a substitute for Render acceptance.

Dashboard/external acceptance still required:

- Confirm Free plan, one service, no disk, successful build and `/readyz` 200.
- Open `/demo`, log in, reload, open trades and inspect SBI 現引/現渡 details.
- Confirm default 30-day Home charts and totals, then verify the date range shown on `/demo`.
- Open news, predictions, stored validation history and performance statuses.
- Verify upload, deletion, password change, tokens and administration are denied
  (including direct HTTP requests), and the data remains unchanged.
- Log out, verify protected reads require authentication, log in again.
- Restart/redeploy: readiness recovers, one dataset returns, old token is rejected.
- Let the actual Render service sleep for at least 15 minutes, then wake it via
  its HTTPS URL; record wake time and verify login/data again. Local container
  restart alone does not prove Render's proxy/sleep behavior.
- Confirm only one public HTTP origin; do not expose the fixture or internal ports.

To remove: delete the web service/Blueprint in Dashboard. There is no external
database or disk to clean up. Recreate from the reviewed commit to restore it.
