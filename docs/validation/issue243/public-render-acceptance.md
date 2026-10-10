# #243 public Render acceptance — 2026-10-08

> Historical acceptance of implementation `70fab644`. The subsequent rolling
> startup-date change requires a new Render deployment and public revalidation;
> see [rolling showcase validation](rolling-showcase.md).

Public URL: https://tradelens-demo.onrender.com

Reviewed branch: `feat/243-render-demo`. Implementation / PR #264 head at the
start of acceptance: `70fab6448842f2117fa3c345d1b2bf4ea8802363`.
Checks started at approximately **2026-10-08 10:09 UTC**. This report is a
subsequent documentation-only change; it does not change the deployed runtime.

**MERGE READY: NO** until the remaining external acceptance gates below are
closed. Public functional and security checks passed. Dashboard configuration
and an owner-controlled redeploy have not been verified. Actual idle recovery
restarted the API, reseeded the fixture and invalidated old refresh tokens. No merge, issue closure,
README advertisement or Render configuration change was performed.

## Implementation review

Reviewed #243, current PR head, `render.yaml`, `deploy/demo/`, deployment/local
validation docs, container acceptance tests, all three existing seed/provider
scripts, demo API/config policy, authentication middleware and affected callers.
The source implements one nginx public listener, private API/provider/readiness
listeners, a fresh SQLite directory per boot, bootstrap-only writable API with
an independent JWT secret, admin demotion, and final fail-closed demo middleware.
Normal Dockerfiles/Compose/GHCR workflows are unchanged in the implementation
commit. No runtime dependency on real brokerage data or live market downloads
was found. This review assumes a small shared evaluation demo, not load-tested
multi-user hosting.

Discrepancies / limits identified before acceptance:

- The earlier local record correctly says external acceptance was pending; it
  is historical evidence, not a claim about this now-accessible public service.
- `/system/info` reports version `0.2.1` but **no commit metadata**. The exact
  deployed SHA cannot be independently tied to PR head through this endpoint.
  Obtain the deployed commit from Render Dashboard before merging.
- Fixed September data is outside the initial 30-day Home chart range. Entry
  instructions say to select All. The trade chart initially requests 15-minute
  bars, outside the daily-only fixture: it displays no data until 1D is chosen.
- Source `plan: free` and local resource limits do not prove Dashboard billing,
  actual instance type, service count, disk configuration or peak memory.

## Summary

| Check | Status | Expected versus actual evidence |
|---|---|---|
| Public HTTPS / certificate | PASS | curl without `-k`: TLS verification result 0; browser loads HTTPS without warning |
| `/` | PASS | HTTP 302, relative `Location: /demo` |
| `/demo` | PASS | HTTP 200 HTML; fiction, read-only, reset and no-private-upload notice; Try Demo link to login |
| `/home` deep link | PASS | HTTP 200 SPA; authenticated render and unauthenticated redirect to login |
| `/healthz` | PASS | HTTP 200 JSON, `status: ok`, version `0.2.1` |
| `/readyz` | PASS | HTTP 200 after initialized fixture; dependency failure behavior is local-test evidence only |
| Entry JavaScript / CSS | PASS | All 81 `/assets/` references in entry HTML fetched with HTTP 200 and JS/CSS MIME types |
| Mixed content / CORS | PASS | Same-origin app/API worked; captured browser warn/error log was empty in exercised flows |
| Login / invalid credentials / refresh | PASS | Direct API login + refresh 200; incorrect password 401; browser invalid-credentials message then successful login |
| Reload / logout / re-login | PASS | Reload retained authentication; logout followed by `/home` redirected to `/login`; re-login restored Home |
| Non-admin user / registration | PASS | `/me` has `is_admin: false`; profile says member; setup status shows one user, no setup or registration |
| SBI / JPY data | PASS | One SBI JPY margin-capable account, 7 closed groups, numeric codes and `285A` render correctly |
| SBI accounting / valuation | PASS | JPY 25,000 realized, 950,000 contributed, 975,000 closing; all six valuation points complete |
| Home / trade charts | PASS | Actual rendered All-range charts; 1D trade chart renders fixture; unsupported intraday range shows explicit no-data state |
| News / predictions / report | PASS | 3 records, 3 predictions, 6 horizons, 4 pending, 2 unavailable, 0 validated; report excludes unvalidated results from hit rate |
| Stored evaluation history | PASS | GET validations for all 3 predictions returned 200; never triggered new public evaluation |
| Backend read-only enforcement | PASS | Forbidden operation matrix returns 403; multipart import refused; before/after dataset snapshots identical |
| Browser read-only error | PASS | News title save refused with explicit read-only message; cancel returns original title |
| Desktop / narrow viewport | PASS | 1280×900 trade-detail rendering and 390×844 viewport override navigation/news/error flows; screenshots below |
| Internal services | PASS | Reviewed source binds API/provider/readiness to loopback; no public fixture route in nginx. Separate-port firewall exposure was not port-scanned |
| Render Free / billing / disk / secret | BLOCKED | Dashboard opened in built-in browser and redirected to `/login`; no authenticated session available |
| Deployed SHA | BLOCKED | Build commit absent from public system info; owner must provide Dashboard commit evidence |
| Public owner-controlled redeploy/reset | BLOCKED | Dashboard requires login and no Render API connector is available; no public restart attempted |
| Actual Render idle/wake / automatic reset | PASS | 967.75 s without agent traffic; first request ready in 23.08 s; API start/user changed; old unexpired refresh rejected; complete duplicate-free fixture returned |
| Public resource peak / load | NOT TESTED | Prior local 512 MiB / 0.5 CPU run passed; no Render metrics or load test |
| Native Expo / other browsers | NOT TESTED | Narrow Web viewport is not native mobile acceptance; only Codex built-in browser used |

## HTTP and data evidence

Initial warm `/home` request: HTTP 200, TLS verify result 0, approximately
0.590 seconds. Responses identify the Render nginx origin through
`x-render-origin-server: nginx/1.22.1`. This measures warm response latency,
not cold startup time. `/healthz`, `/readyz` and `/` were sampled at
10:10:12–10:10:13 UTC.

Reused the assertion-based `check()` function from `scripts/test_demo_render.py`
against the public HTTPS base, without running its local Docker/restart section.
It completed successfully in 36.9 seconds. Tokens remained only in temporary
local files and were never included in committed evidence. Public identity is
represented by the SHA-256 prefix `ab1f7fdedb34`, not credentials or tokens.

| Symbol / group | Realized JPY |
|---|---:|
| 6501 cash long | 10,000 |
| 285A margin long | -5,000 |
| 7203 margin short | 10,000 |
| 6758 margin leg ending in 現引 | 0 |
| 6758 converted cash leg | 5,000 |
| 8306 cash leg ending in 現渡 | 5,000 |
| 8306 margin short ending in 現渡 | 0 |

Final valuation point (2026-09-08): estimated value 975,000; contributed capital
950,000; cash 975,000; open value 0; realized 25,000; unrealized 0; complete;
no warnings. The conversion leg is not double-counted as additional profit.
The browser displayed `信用转现物` and the daily chart conversion marker for
6758. Public fill details also retained `sbi:cash`, `sbi:margin-long` and
`sbi:margin-short` lots. Both 現引 legs share the conversion identity, and the
cash leg carries transferred cost basis 150,000. Both 現渡 legs share a
settlement identity, record disposed cash cost basis 90,000 and settlement
realized P&L 5,000. Group P&L remains 5,000 on the cash leg and 0 on the short
leg, avoiding double counting. Unsupported intraday
bars showed an explicit no-data state, rather than fabricated bars.

News browser report explicitly displayed total 6 / pending 4 / validated 0 /
unavailable 2 / incomplete 0 and hit-rate `n=0`. These are stored fictional
evaluation states, not validated investment results. Public GET history returned four stored evaluation records: two pending and
two unavailable; the never-evaluated 6501 prediction has no evaluation history.
Recalculation is intentionally forbidden.

## Security evidence

The reused container check's public function asserted HTTP 403 for setup,
registration, account creation/deletion, trade edits, password/TOTP changes,
token listing/creation/revocation, imports/commit, media/attachments, OCR,
admin reads/writes, AI configuration/analysis, shares and integration routes.
Unauthenticated account reads returned 401. System feature flags enable only
fictional market data among the tested external-integration features.

Additional real-route probes are recorded in [public-security.txt](public-security.txt):
execution creation, trade deletion, account update, profile mutation, OCR parse,
AI configuration read, broker sync read/run, trade coaching, password mutation,
and multipart SBI CSV import all returned 403. Payloads were empty or fictional;
no real brokerage statement was uploaded. Before and after snapshots of
accounts, trades, executions, news, cash flows, user and preferences were equal.
Registration/setup denial happens at the backend even without a logged-in user.
Email mutation has no normal profile-edit route; its attempted PATCH also fails
closed. No public credentials, secrets, session tokens or sensitive response
bodies are included in this report.

## Browser evidence

Screenshots are from the public HTTPS service, not the previous localhost run:

- [Home charts and JPY P&L](public-home.jpg): All ranges selected, real rendered
  equity and account-value charts. Default 30-day empty state was also exercised.
- [Narrow viewport Home](public-mobile.jpg): requested browser viewport 390×844;
  browser DOM confirmed width 390. Screenshot transport captures 375×812, so
  screenshot pixel dimensions alone are not the viewport measurement.
- [News performance](public-news.jpg): pending and unavailable remain separate
  from validated outcomes.
- [Read-only rejection](public-read-only.jpg): news edit submitted, backend
  refusal shown, cancel restored the original title.
- [Desktop SBI detail](public-sbi.jpg): 1280×900, 現引 marker on the 1D chart.
- [Narrow viewport trade](public-mobile-trade.jpg): 8306 short leg explicitly
  displays 現渡; narrow-viewport login and logout were also exercised.

Normal editing/export/AI/upload controls remain visible. Their submissions are
blocked by the backend; UI hiding is not the security boundary. Restricted
settings/import history can show empty/unavailable states. These are documented
demo UX limitations, not evidence that those operations are enabled.

## Actual idle recovery and reset

Baseline final request: **10:14:30.774 UTC**. All agent-created demo tabs were
closed, and no Demo HTTP polling/keep-alive occurred for **967.75 seconds**.
The first wake request to `/readyz` started at approximately **10:30:38.524 UTC**
and returned HTTP 200 with the expected empty body after **23.08 seconds**.
API `started_at` changed from `09:58:47.853182957Z` to
`10:30:58.198663715Z`, and the demo user identity changed. An old, still-valid
30-day refresh token returned HTTP 401. Fresh login and the full public
fixture/auth/permission check passed again: one user/account, seven closed
trades, three news/predictions and the expected valuation/outcomes.

[Machine-readable timing and reset evidence](public-idle-wake.json). This is
observed recovery after the actual public service idle interval, consistent
with Render Free sleep. Without Dashboard events we cannot independently
exclude an unrelated restart or certify the precise platform sleep cause.
We did not control other visitors. No manual restart/redeploy was attempted.
The observed restart does independently establish clean automatic reinitialization
and stale-refresh rejection on the public service. The browser also redirected
the old session to login after recovery; fresh narrow-viewport login succeeded.

## Remote CI

For implementation head `70fab6448842f2117fa3c345d1b2bf4ea8802363`, GitHub
reported all four checks completed with SUCCESS:

- [Test API](https://github.com/HoesenBruce/TradeLens/actions/runs/37758348164/job/113248389597)
- [Test web](https://github.com/HoesenBruce/TradeLens/actions/runs/37758348005/job/113248388173)
- [PR title](https://github.com/HoesenBruce/TradeLens/actions/runs/37758348035/job/113248387732)
- [Second PR-title trigger](https://github.com/HoesenBruce/TradeLens/actions/runs/37758348065/job/113248389761)

The duplicate title checks are separate workflow triggers. This is remote CI
evidence for the implementation SHA, not a prediction about later commits.

## Findings and remaining steps

**BLOCKER — acceptance evidence:** owner must confirm Dashboard Free instance,
one Web Service, no disk/paid resources, free-hour availability, generated random
JWT configuration (without disclosing its value), and the deployed SHA. A public
successful response cannot prove any of these account settings.

**NON-BLOCKER — manual lifecycle control:** actual public idle recovery/reset
passed. Owner-controlled Manual Deploy/restart is still NOT TESTED because
Dashboard access is unavailable. If required for operational sign-off, repeat
the same identity/count/refresh checks around an owner-triggered redeploy.

**NON-BLOCKER — demo UX:** fixed date/range defaults and visible write controls
require reading the entry instructions; backend refusal is clear. These do not
alter self-hosted UX or accounting behavior.

**OPTIONAL IMPROVEMENT — build traceability:** populate existing build commit
metadata in the isolated Docker build, or retain Dashboard deployment evidence
for every accepted SHA. No new public endpoint is necessary.

No independently verified runtime defect currently requires a code fix. Keep
PR #264 draft, record the missing Dashboard/deployed-SHA evidence, then
recheck CI for the final head and mark ready for review. Merge remains a separate
human action; Issue #243 stays open until its acceptance gates are met.
