# Issue #316 — calendar holding duration

Validated on 2026-10-15 with fictional executions and a disposable SQLite API.

## Behavior and data trace

- Generic/SBI import retains `source_time_precision: date`. Existing source timestamps and importer anchoring remain unchanged.
- Aggregation, accounting strategies, trade boundaries, fill IDs, P&L, fees and stored `time_in_trade_secs` are unchanged.
- The shared list/detail `holding` read model matches the aggregated endpoints to linked executions. It returns elapsed seconds for precise endpoints, market-local calendar days for either date-only endpoint, or unknown for missing/invalid endpoints and incomplete opening evidence.
- Calendar days compare civil dates, including zero-day holds, across midnight and DST. Open positions use the current date in the existing instrument-market timezone resolver and carry `ongoing: true`.
- List rows, compact lists, the peek drawer and full details use the same localized formatter. Accessible descriptions distinguish calendar days from exact elapsed time. The SBI caveat is refined on new aggregation and when reading legacy warnings; no historical regroup is required.
- The existing #307/#309 direction fix and #308/#310 quantities model are preserved. The quantity-card clipping reported in #317 is a separate follow-up.

## Verification

- Go unit coverage: same-date/next-date/three-day, either mixed endpoint, precise 09:30→10:45, UTC-vs-Tokyo/New York midnight, spring/fall DST, open positions, missing metadata, unknown precision and precise endpoints with interior date-only fills.
- API persistence/read-back compares list/detail holding metadata and unchanged monetary fields. Generic importer regression confirms date precision without an SBI offset.
- `go vet ./...`, `go test ./...`, `go build ./...` pass.
- Web check passes (0 errors; existing lint warnings remain), all 199 test files / 1,242 tests pass, and TypeScript plus production build pass. CI results are recorded on the PR.
- Playwright: `E2E_BASE_URL=http://localhost:5174 PLAYWRIGHT_BROWSERS_PATH=/tmp/tradelens-browsers ./node_modules/.bin/playwright test e2e/trade-holding.spec.ts` passes against the real API on :8092. It covers actual SBI and generic CSV imports, precise/mixed endpoints, 0/1/3 days, ongoing short, incomplete opening history, list/peek/full detail, close/reopen/reload, English/Chinese/Japanese, a Los Angeles browser timezone and 390px Web layouts. Persisted trade/monetary fields remain unchanged after viewing.
- Screenshots were visually inspected. Built-in browser tools were unavailable, so Playwright Chromium was used. Market-data and calendar feeds were disabled in the disposable API; chart/provider behavior is outside this change.

## Evidence

- [Desktop list](holding-list.png)
- [Peek drawer](holding-peek.png)
- [Chinese full detail](holding-detail-zh-CN.png)
- [Compact detail](holding-compact.png)
- [Compact list](holding-compact-list.png)

Mobile native apps were not validated; outside current fork scope. Older servers without `holding` retain the legacy exact-duration fallback.
