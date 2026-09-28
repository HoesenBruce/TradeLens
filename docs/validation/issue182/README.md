# Issue 182 — grouped and trade-row analytics

## Contract

Uses the Summary/Equity shared latest-FX seam. Explicit `target_currency` on Trades and Breakdown returns metadata plus `trades` / `groups`; no target retains the native legacy array and mixed-scope guard. Behavior and Monte Carlo add the same currency metadata. Trade list monetary P&L, fees, prices and initial risk are display-normalized; `source_pnl_currency` records provenance. Native trade detail, exports and stored accounting are unchanged. Monte Carlo `ruin_threshold` is in output currency. R-multiples divide each trade's own P&L by its own risk before aggregation and do not need FX.

## Local checks

- Go full suite passed; grouped/trade tests cover JPY+USD to JPY and USD, All Accounts, single account identity, unavailable FX, native detail preservation, deterministic simulation scaling and mixed R endpoint.
- Web full suite: 179 files / 1,031 tests passed. Focused hooks/Playbook rerun after final failure-state adjustment.
- Web check: zero errors (74 pre-existing warnings); production build passed.

## Real browser acceptance

Built-in browser, current-tree API on 8097, throwaway SQLite database, Web on 5186. Latest live Yahoo FX (not historical FX; displayed amounts vary slightly as quotes refresh).

- Calendar month/week: two original winners, separate converted daily amounts, PF/expectancy/fees/counts and day drawer; converted JPY list row opens a native JPY detail, including executions.
- Year mode: September count 2 and summed JPY P&L; month/year both directions exercised through re-entry.
- Trades: JPY and USD monetary columns, preserved percentage/R ratios; single JPY account shows native 200 and five rows after adding break-even samples; All Accounts restores ten rows.
- Reports: normalized Setup breakdown total; behavior report; ten mixed samples enable Monte Carlo rather than stopping at its insufficient-data state.
- Playbook: Currency QA groups the JPY and USD winners, showing summed output P&L and expectancy.
- Wrapped: ten samples, USD total 21.27, best day USD 20, JPY counterpart around 3,349; rendered at 390px.
- Home contribution: USD account 20; JPY account about 1.27 in USD, not raw 200.
- FX failure: disposable ZZZ-source account makes real API return `fx_unavailable`; Playbook hides monetary statistics, Trades shows error and no rows. Remove disposable fixture, click Retry, ten rows recover.
- Currency menu readback and account reset/re-entry exercised. No new commit/cancel form in this part.

## Boundaries

Header funding return/Reports percentage denominator, annual goal and explicit mixed subsets remain #183. Currency-less monetary risk limits remain fail-closed for mixed Compliance/Execution Score; no currency is guessed. Historical account value, accounting, replay and cost basis were not edited. Mobile not validated, outside current fork scope.
