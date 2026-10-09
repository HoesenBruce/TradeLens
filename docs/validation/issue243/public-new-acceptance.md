# Updated Render public acceptance — 2026-10-08 UTC

URL: https://tradelens-demo.onrender.com

Target runtime revision: `3e8f7b8090673ba3d23663f2d47eecfbf590dc86`
(PR #264). User reported deployment. Public system info reports version 0.2.1,
not Git SHA; exact deployed commit and Free/resource configuration require
Dashboard evidence. Observable fixture/entry behavior matches the target update.

## Independent HTTPS/API acceptance

The production image is not inferred from local tests. The existing container
acceptance check's read/auth/permission/SBI/news assertions were independently
run against this HTTPS origin, excluding Docker-only initialization controls.

- GET `/healthz` and `/readyz`: 200. HEAD `/readyz`: 501 (the readiness handler
  implements GET; HEAD is not used as readiness evidence).
- `/demo`: fictional calendar/107-trade notice; generated window
  **2026-09-09 – 2026-10-08** (Tokyo October 9 startup).
- First-user setup closed, registration closed, one non-admin demo user.
- Correct login/refresh work; bad credentials and unauthenticated reads rejected.
- 107 closed trades, both long/short, varied profit/loss; realized JPY 25,000.
- Complete daily historical account valuation; closing JPY 975,000.
- Calendar test range: 42 fictional entries over 14 days; JPY/high filter yields
  14; GBP filter empty; invalid dates rejected; unauthenticated read rejected.
- Three news items and three predictions/six horizons, four pending and two
  unavailable; stored validation history readable. Existing SBI outcome retained.
- Existing deny cases for setup/registration, writes/deletion, uploads, password
  and TOTP changes, API tokens, sensitive administration, AI/OCR, broker sync,
  sharing and attachments return 403. Trade count remains 107 afterwards.
- [Sanitized API evidence](public-new-api.json). No login tokens or secrets in
  this report or repository.

## Built-in browser

- Entry and login; Home shows 107 trades and populated default 30-day charts.
- Calendar displays explicit fictional titles; high filter excludes EUR/USD;
  reopened filter shows high selected; Escape preserves it. USD+high produces
  the empty-filter state. Clearing restores the list; next week loads events.
- Reports Win/Loss: windows 10, 20, 50 and 100 render (latest rates 50%, 50%,
  52%, 51%). At 100, long-only correctly shows insufficient data; all-directions
  restores the curve. Returning to 10 and refreshing restores the populated
  default view. Screenshots inspected visually:
  [calendar](public-new-calendar.jpg), [10-trade](public-new-rolling-10.jpg),
  [100-trade](public-new-rolling-100.jpg).

## Idle recovery

PASS: closed the agent's public tab and stopped API traffic at
2026-10-08T15:46:21.535597Z. First request after **971.49 seconds** without agent
traffic recovered GET /readyz in **22.82 seconds**.

- API started_at changed from `2026-10-08T15:41:30.391813449Z` to
  `2026-10-08T16:02:47.654645493Z`; the demo user identity also changed.
- Old refresh token (expiry November 7, still unexpired) returned HTTP 401.
- Fresh login and the complete public read/auth/permission/calendar/SBI/news
  checks passed again, including 107 closed trades, JPY 25,000 realized and
  JPY 975,000 closing value. No duplicate dataset after reseeding.
- Built-in browser with the previous session redirected to login after wake;
  fresh login restored the populated default Home charts and 107 trades.
  [Post-wake Home screenshot](public-new-wake-home.jpg).
- [Sanitized idle/recovery evidence](public-new-idle-wake.json).

This independently demonstrates restart/reseed and recovery following the idle
interval, consistent with Render Free sleep. The platform's exact restart cause
still requires Dashboard logs; agent cannot exclude unrelated platform actions.
Refresh tokens were held only in a private temporary file and are not committed.

## Dashboard and scope

Exact deployed SHA, Free plan/no disk/no paid resources remain unconfirmed by the
agent. Dashboard requires login. Manual restart not performed. Native mobile,
other browser engines and peak/load behavior not tested. README unchanged;
no merge or issue closure performed.
