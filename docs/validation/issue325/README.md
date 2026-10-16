# #325 contextual metric help

Base: `8adee59ae642ea514d25e3f863fdcef10b7c308b` (merged #324 / PR #326).
Branch: `feat/325-contextual-metric-help`.

## Coverage and authoritative sources

Definitions live in `web/src/lib/metrics/definitions.ts`; all 18 IDs have five localized
fields in en, zh-CN, zh-HK, ja and ko. Only explanations and their controls changed.

| Metric / surface | Computation sources checked |
| --- | --- |
| Holding, list/detail/preview | `api/internal/api/trade_holding.go`, `web/src/lib/tradeHolding.ts`, holding regression tests |
| Opened/closed/remaining and position, list/detail/preview | `api/internal/api/trade_quantities.go`, `web/src/lib/tradeQuantities.ts`, quantity readback and progress tests |
| Gross/net realized P&L and fees, list/detail/preview/home/reports | `api/internal/trades/engine.go`, `sbi_margin.go`, `split_boundaries.go`, `web/src/lib/tradeInsights.ts`, `api/internal/api/analytics_handlers.go` |
| Trade return % | default `openState.finalize`, SBI settlement overrides in `sbi_margin.go` / `split_boundaries.go` |
| Planned R:R and realized R, list/detail/preview | `web/src/lib/riskReward.ts`, `tradeColumns.tsx`, `api/internal/api/trade_detail.go` |
| Win rate / profit factor / payoff, header/home/reports/tables | `api/internal/analytics/analytics.go`, `api/internal/api/dto.go:loadClosedTradeRows`, `web/src/components/ReportsSummaryBento.tsx` |
| Equity drawdown amount and range-local % | `api/internal/analytics/analytics.go:EquityCurve`, `api/internal/api/analytics_handlers.go:handleEquityCurve`, `web/src/lib/reportsAnalytics.ts:drawdownSeries` |
| Account return, header/account details | `web/src/lib/headerStats.ts`, `web/src/app/screens/AccountDetailView.tsx`, `settings/settings-sections.tsx:ledgerBalance` |
| Report net/gross, funding %, fee-% base | `ReportsDisplayContext.tsx`, `ReportsSummaryBento.tsx`, `web/src/routes/reports.tsx`, `web/src/lib/headerStats.ts:netDeposits` |
| Period averages/annualized rate | `web/src/lib/reportsAnalytics.ts:periodReturns`, `ReportsPeriodReturns.tsx` |
| Rolling win rate / metric evolution | `web/src/lib/reportsAnalytics.ts`, `web/src/lib/homeInsights.ts:chronologicalClosed` |
| Simulated maximum drawdown | `api/internal/analytics/montecarlo.go`, `ReportsMonteCarlo.tsx` |

## Existing conventions made explicit

- Any date-only boundary uses market-local calendar days (same date 0, next date 1).
  Precise timestamps use elapsed time. Missing boundaries stay unknown.
- Quantity is the authoritative read model: closed = opened − remaining; normalized
  progress is closed/opened. Long/short quantities are magnitudes; reversal lifecycles
  remain separate. Incomplete SBI opening evidence and unverified units stay unknown.
- Default open/partially closed trade P&L remains unavailable; existing SBI partial
  realized amounts remain available where supplied. Summary eligibility still requires
  closed status and known P&L. Breakeven counts in the win-rate denominator.
- Profit factor uses after-fee winning/losing sums despite the API's gross bucket names.
  No losses returns API value 0 (header 0.00; some cards —), not infinity. The separate
  payoff ratio can display infinity; mean/median modes use their corresponding samples.
- Reports' auxiliary **Gross** row currently sums positive net P&L plus absolute negative
  net P&L. It differs from the net/gross-controlled hero. The fee-% base uses the former.
  This inherited result is explained, not changed.
- Account % uses the signed cash ledger, whose opening deposit already includes opening
  balance. No cash-flow timing adjustment, compounding, TWR or MWR is added. Header and
  account details retain their distinct eligibility/nonpositive-denominator behavior.
- Equity drawdown includes cash flows, so withdrawals can cause declines. Range-local
  drawdown % uses visible peaks; simulation drawdown uses cumulative net P&L without
  cash flows. Funding-based report % is a different denominator.
- Realized R uses journal initial risk, excluding dividends. The list requires positive
  risk while detail accepts nonzero risk; planned target/stop R:R is a separate ratio.

## Omitted indicators / scope boundaries

The current Position column is a quantity, not a live currency exposure value. No new
market-value exposure, unrealized return, finance indicator, calculation trace or knowledge
base was added. Existing entry/exit totals are not redefined as remaining exposure. Only
currently rendered in-scope metrics receive controls; optional legacy risk cells keep their
current visibility rules. Mobile apps remain preserved and unvalidated, outside fork scope.
No API, accounting algorithm, read model, importer, schema or persisted-data behavior changed.
No release, Release Please merge or deployment is included.

## Acceptance

Disposable API: `/tmp/metrics-qa/qa.db` on port 8092 with explicit local Web CORS origins.
Browser Web port 5174; all users/accounts/fills are synthetic. A working `/healthz` and actual
API readback were checked; tests do not mock finance responses.

- Focused initial component tests: 7 files / 41 passed.
- Additional catalog/report/format regression tests: 10 files / 49 passed. All definition
  fields resolve; non-English caveats are checked against English to detect silent fallback.
- `trade-holding.spec.ts` and `trade-quantities.spec.ts`: both passed after integration,
  including SBI imports, missing history, long/short, reversals, corporate-action conversion,
  fractional/large quantities, sorting, full detail/preview and narrow layouts.
- `contextual-metric-help.spec.ts`: real API seed, exact before/after trade readback,
  known net/gross/return/fees, partial and short progress, reversal split, date-only vs precise
  duration, win/loss/breakeven denominator, account funding once, report tabs, preview nesting,
  five-locale detail and report narrow views, touch, keyboard, re-entry and close/focus.
- Boundary scenario: zero funding and 12 winning trades retains PF=0, payoff=∞ and no
  funding-% control; simulation explanation confirms cumulative P&L rather than equity.
- `metric-help.spec.ts`: shared foundation desktop/narrow keyboard, touch and localization.
- Built-in browser: real API login, list header keyboard/open/close/re-entry, 390px card
  quantity help and visual inspection. Playwright screenshots also inspected.
- No apply/cancel/reset workflow exists for these read-only explanations. Dismissing them
  changes neither filters nor trade state; sorting and nested preview preservation are tested.

Static check: 0 errors (347 existing warnings); TypeScript and production build passed.
Full local suite: 202 files / 1,253 tests passed (`vp test run --maxWorkers=2`).
Final shared/business E2E: 4 passed; existing holding/quantity E2E: 2 passed. Fresh PR CI is
required for the final source revision. Screenshots below capture representative states;
all five locales are exercised by E2E.

[Desktop reports](reports-desktop.png) · [Chinese detail](detail-zh-CN-narrow.png) ·
[Chinese reports](reports-zh-CN-narrow.png) · [Compact list](list-narrow.png) ·
[Zero-loss simulation](zero-loss-simulation.png).
