# Demo calendar and rolling statistics acceptance

Follow-up to #243 / PR #264. Supersedes the seven-trade fixture count in the
previous rolling-date report; previous public acceptance remains historical.

## Changes

- Permit authenticated demo GET /economic-events and serve the seeded SQLite
  archive without a configured/live economic feed. All other demo restrictions
  remain enforced. Ordinary self-hosted calendar behavior is unchanged.
- Seed 135 explicitly fictional economic events covering startup -30 through
  +14 days: JPY/high, USD/medium, EUR/low, synthetic forecasts/previous values,
  actuals for past dates only. Navigation outside this range yields no events.
- Add 100 independent synthetic SBI round trips only in the rolling Render
  fixture, using the existing importer and loopback price provider. There are
  now 107 closed trades, including 50 additional long and 50 short trades,
  balanced additional profits/losses, unchanged net P&L JPY 25,000 and final
  account value JPY 975,000. Fixed standalone fixture remains unchanged.
- Entry page explicitly explains the calendar and rolling statistics coverage.

## Local evidence

- `python3 scripts/test_showcase.py`: PASS (including fixed defaults, rolling
  dates, 212 CSV rows, 100 extra symbol price fixtures and calendar boundaries).
- `go test ./internal/api -count=1`: PASS (28.3 seconds). Demo policy and
  economic-event regressions include local archive with
  no provider, authentication, invalid input, filtered results and forbidden POST.
- Final Linux ARM64 demo Docker image built successfully.
- `python3 scripts/test_demo_render.py`: PASS on the final image, under 512 MiB
  / 0.5 CPU. [Output](calendar-stats-container-test.txt). Two clean boots each
  verify 107 closed trades, varied direction/profit/loss, complete valuations,
  SBI/news outcomes, calendar dates/filters/empty results/authentication, existing
  write restrictions and stale-token rejection. Provider death still stops the
  service. No duplicates across reset; original totals retained.
- Codex built-in browser, actual local container/API: login; calendar current
  week showed fictional events; high-impact filter excluded EUR/USD, reopening
  showed selected high, Escape preserved state, URL re-entry read it back.
  USD+high yielded the empty-filter state; clearing both restored the list.
  Next week and previous week, then refresh, loaded correctly.
- Reports > Win/Loss: all-direction windows 10 -> 20 -> 50 -> 100 displayed
  curves. Selecting long at 100 correctly showed insufficient data; restoring
  all directions restored the curve. Returning to 10 and refreshing produced
  the populated default. Screenshots inspected visually:
  [calendar](calendar-events.jpg), [rolling statistics](rolling-107-trades.jpg).

## External acceptance

This revision has not been deployed or tested on Render. Owner must Manual Deploy
this PR revision in the Dashboard, confirm the deployed SHA and Free/no-paid
resource settings, then repeat calendar/statistics and cold-start acceptance.
Native mobile is outside scope and untested. Direction/duration/date filters may
legitimately leave fewer trades than larger rolling windows require.
