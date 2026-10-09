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

Reset via Render's restart control, or redeploy an explicitly recorded approved
release commit. After release deployment activation, avoid **Deploy latest commit**:
it can substitute unapproved main source for the pinned release.
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
expiry. The fixed standalone showcase has seven closed trades; the Render
fixture has 107 (see below). Both produce JPY 25,000 realized P&L; closing
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

## Calendar and rolling statistics

Render seeds 107 closed fictional trades: the original seven SBI groups plus
100 independent synthetic cash/margin round trips (50 long, 50 short), spread
across the startup-relative window. Additional trades net to zero, preserving
JPY 25,000 total realized P&L and JPY 975,000 closing value. Codes and prices
are fictional exercises, not claims about actual securities. Standalone showcase
commands retain the original seven groups unless the runtime date is selected.
The all-directions view supports windows 10, 20, 50 and 100; narrower filters can
legitimately have too few trades for larger windows.

Economic events are seeded locally for startup day minus 30 through plus 14 days,
with JPY/high, USD/medium and EUR/low entries, explicitly titled `[FICTIONAL demo]`.
Past entries include synthetic actuals; future entries do not. The demo API reads
this archive with normal authentication, date/impact/currency filters and input
validation. It never contacts a live calendar feed. Outside this bounded range,
the calendar correctly shows an empty result. Every service restart regenerates
both trades and calendar entries; deploying this change requires Render's manual
deploy of the latest PR revision.


## Release-gated deployment (#286)

Implementation is ready for offline review; **no live deploy, secrets setup or
activation is claimed**. `DEMO_AUTO_DEPLOY_ENABLED` must remain unset/false until
owner-authorized controlled acceptance succeeds. Public service settings were
not accessed or changed by this implementation.

The chain is Release Please → reusable Docker publisher → both matrix image
jobs succeed (including API/Web CI and `ghcr` approval) → reusable
`demo-deploy.yml` → `render-demo` approval → validation → Render API → Live SHA
verification → bounded public smoke. It does not depend on `release: published`:
GITHUB_TOKEN event suppression cannot break a direct reusable-workflow call.
An ordinary main push produces no release and therefore never calls publication
or Demo deployment. Prereleases/SHA-only builds do not call Demo deployment.
A manual publisher running on a tag does not automatically deploy Demo; use the
main-based guarded Demo recovery dispatch after successful publication.

The deploy script accepts only published, latest stable `vX.Y.Z` releases on
main, with matching immutable full SHA, VERSION and release manifest. It requires
two non-expired digest artifacts from the trusted successful publisher/Release
Please run for that exact source/ref/version, then anonymously checks both GHCR
version and full-SHA tags against those digests. The automatic call can inspect
its still-running parent because both publisher jobs and artifact uploads have
already succeeded. Manual recovery requires a completed successful run.
Missing, failed, rejected, partially published, expired or mismatched evidence
fails closed. An arbitrary SHA/branch is never a target input.

The release's own `deploy/demo/smoke.json` supplies fixture expectations. Schema 1
checks 107 closed trades / 214 executions, JPY 25,000 P&L, contributed capital
950,000, account value 975,000 and three fictional news items. Update this contract
alongside intentional fixture changes. Releases without this contract and the
Demo Dockerfile are rejected: **v0.2.1 cannot serve as the first acceptance target**.
The script also checks `/demo`, authentication, non-admin user, fictional calendar,
backend `demo_read_only` responses for funding/import/security/AI/admin writes and
reads, and unchanged trades after attempted writes. Public requests occur only
in the bounded post-deploy smoke, never as periodic keep-alive traffic.

### Owner setup (after Draft review, before controlled acceptance)

1. GitHub → repository Settings → Environments → create **render-demo**. Select
   **main** as the only allowed deployment branch; require HoesenBruce review,
   disable administrator bypass. Single-owner self-review may stay enabled.
   Protect main with PR/review/check requirements; never expose secrets to PR jobs.
2. In that environment, add these **environment secrets**, not repository-wide
   credentials:

   | Name | Value / source |
   |---|---|
   | `RENDER_API_KEY` | Render Account Settings → API Keys → create a dedicated key for this automation; paste it directly into GitHub's secret input. |
   | `RENDER_SERVICE_ID` | Existing `tradelens-demo` service's `srv-…` ID from its Dashboard URL (20 lowercase alphanumeric characters after `srv-`). |

   The API key needs service/deploy read access and permission to trigger deploys.
   Render documents account keys as accessing every workspace the account belongs
   to, not single-service tokens. Use the least privileged available existing
   account with Demo access, rotate/revoke the dedicated key, and review the actual
   credential scope before storing it. Do not add paid memberships/resources to
   achieve narrower scope. The script only reads service/deploy data and POSTs
   deployments; it never reads/writes environment secrets or changes resources.
3. GitHub's built-in `GITHUB_TOKEN` needs **contents: read** and **actions: read**
   to validate releases and download digest evidence. These are declared through
   all reusable callers. Existing package write stays confined to publication;
   no PAT/dispatch token or extra package write is needed for Demo.
4. Read back the existing Render service: repository HoesenBruce/TradeLens,
   branch main, one Docker **Free** Web Service, root/context `.` (or blank),
   `deploy/demo/Dockerfile`, no command override/pre-deploy command, health
   `/readyz`, URL `https://tradelens-demo.onrender.com`, no disk/autoscaling and
   **Auto-Deploy Off**. The script refuses a different configuration rather than
   changing it. Retain the generated `TM_JWT_SECRET` in Render; never copy it to
   GitHub. Blueprint management and the single service remain intact.
5. Leave repository variable **DEMO_AUTO_DEPLOY_ENABLED unset/false**. No real
   secrets are required for Mock CI. Do not set the variable merely to merge code.

A deploy hook is deliberately unnecessary. A hook alone cannot establish actual
Live completion/source; authenticated API access is needed for status anyway.
Using the documented API `commitId` avoids storing a redundant secret or mishandling
hook query strings. If a legacy hook exists (Dashboard → Settings → Deploy Hook),
leave it unused; never paste its URL into logs or records. API deployment does
**not** turn off Auto-Deploy, so the workflow verifies it is already Off both
before and after deployment.

### Controlled acceptance and activation

After this workflow is reviewed and merged, obtain separate owner authorization
for one redeploy. Choose a newly approved latest stable release containing the
Demo and smoke contract, with a successful publisher run and both digest artifacts.
Do not create/merge a release solely to complete this issue without release authority.

GitHub Actions → **Deploy Render demo** → Run workflow → branch **main**:
set `version` to the bare stable version and `publication_run_id` to its successful
publication run ID. Only for the first transition from the pre-release feature
branch Demo, set `bootstrap_live_sha` to the exact current Dashboard Live SHA
separately reviewed by the owner. This pins the source being replaced; it can
only replace a lower VERSION and cannot deploy an older release over a newer one.
Leave it empty on normal retries. Review the environment approval after inputs
are visible, then approve only this controlled attempt.

Record text only: workflow URL, release/version/SHA, both publication digests/run,
Render deploy ID, started/finished timestamps, actual Live commit, Free/no-disk/
Auto-Deploy-Off read-back, and smoke results. The run summary and 90-day
`render-demo-<run>-<attempt>` artifact contain a sanitized JSON record. Copy it into
a durable validation record; do not upload Dashboard screenshots. Independently
open public `/demo`, log in and confirm Web routes. A trigger 201/202 alone is
not acceptance. Deployment polling is capped at 30 minutes; public readiness at
5 minutes; the whole job at 45 minutes. A failed smoke marks the workflow failed
even when Render has already promoted the source.

Only after this controlled run passes and its record is reviewed, owner sets
repository variable `DEMO_AUTO_DEPLOY_ENABLED=true`. Keep `render-demo` review
protection unless the owner separately approves a policy change. Validate one
subsequent legitimate Release Please release-chain run before closing #286;
offline mocks do not prove GitHub approvals, nesting, registry access or Render
configuration. No such run is claimed by the implementation PR.

### Races, retry and recovery

All automatic and manual Demo invocations share one concurrency group with
`cancel-in-progress: false`, including environment approval wait and smoke.
GitHub may replace a pending run with a newer pending run; it does not guarantee
FIFO. Every run revalidates latest stable, tag/SHA, publication evidence and current
Live immediately before deployment. An obsolete approval therefore fails closed.
The Live source must be an ancestor of the intended release (or the explicit
older-version bootstrap described above). A newer main/release or divergent
Live source blocks deployment. Existing queued/building/updating Render deploys
also block retries; a runner timeout does not mean Render canceled the build.

Avoid out-of-band Render deployments and release/tag mutation during a run.
GitHub concurrency cannot lock Dashboard operators, external API callers or tag
administrators. Configure stable tag protection and restrict access to those
paths. A new release while a build is underway waits for the existing run; the
next attempt resolves the current latest stable. Before/after Live-ID checks catch
interference, but these remote checks are not an atomic cross-provider transaction.

| Failure | Action |
|---|---|
| Approval rejected, publication failed/missing/partial | No Render trigger. Inspect publication and repair it under the existing release policy; never bypass the two-image gate. |
| POST timeout/network failure | May already have created a deploy. Inspect Render history first; no automatic POST retry. |
| Render build failed, canceled or timed out | Inspect exact deploy ID/history; wait for any ongoing deployment. Record the last verified Live SHA; do not infer rollback. |
| Live SHA mismatch or smoke failed | Workflow fails. Record actual Live SHA and diagnose the failed probe; Render may already be serving the new release. |
| Safe retry of current release | Dispatch main with the same validated version and successful publication run ID after Render is idle; reapproval and all checks still apply. |
| Digest artifacts expired | Fail closed. Recover verified evidence through a separately reviewed recovery change or a new approved release; do not republish immutable tags blindly. |
| Roll back to an older known-good release | This workflow intentionally rejects old releases. Obtain explicit rollback authority, use Dashboard **Deploy a specific commit** for the recorded previously good SHA, retain Auto-Deploy Off, wait for Live, then smoke using that release's contract and record the outcome. Never use deploy-latest. |

There is **no automatic rollback**. A redeploy recreates the ephemeral fictional
SQLite database and invalidates sessions as designed. Build-minute/free-hour
allowances remain the owner's responsibility; do not upgrade the plan to recover.

### Offline regression checks

```sh
python3 -m unittest discover -s scripts -p 'test_*deploy*.py'
python3 -m unittest discover -s scripts -p 'test_docker_release_metadata.py'
actionlint .github/workflows/demo-deploy.yml .github/workflows/docker-publish.yml \
  .github/workflows/release-please.yml .github/workflows/api-ci.yml
```

Deployment tests are included in API CI and the publisher's reused API CI. Mock
Render + public API tests exercise exact POST body, credential separation,
completion, fixture smoke, write rejection, failed builds, timeout/mismatch,
release changes and invalid configuration. Local container smoke is useful extra
evidence; it does not satisfy the controlled live acceptance requirement.

References: [Render pinned-commit deployment](https://render.com/docs/deploys#deploying-a-specific-commit),
[trigger API](https://api-docs.render.com/reference/create-deploy),
[retrieve deployment](https://api-docs.render.com/reference/retrieve-deploy),
[API credential scope](https://api-docs.render.com/reference/authentication),
[GitHub token events](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow).
