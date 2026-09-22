# Analytics metric definitions

This document is the canonical index for the analytics shown by TraderMemos.
It describes the current implementation; changing a formula requires changing
the implementation, its tests, and this document together.

## Scope and common rules

- Unless stated otherwise, analytics use **closed trades** after the selected
  account, date, symbol, instrument, tag, side, and duration filters are applied.
- `net_pnl` is realized P&L after recorded fees. Open positions do not enter the
  trade summary, daily P&L, breakdown, R, behavior, compliance, execution-score,
  or Monte Carlo datasets.
- Money values use the account currency. A portfolio response is valid only for
  accounts with the same currency; TraderMemos does not perform FX conversion in
  the API.
- Server summary money values are rounded to two decimal places. Ratios are
  fractions unless their field or table below explicitly says “percent”.
- A zero can mean either a genuine zero or “undefined” where the API contract has
  no nullable field. The boundary behavior is documented per metric below.
- Calendar-day grouping uses the requested trader/market timezone. Session and
  duration classifications are the exceptions documented below.
- Paper/backtest accounts are excluded from portfolio-wide analytics, but are
  included when explicitly selected.

Implementation entry points: [`analytics_handlers.go`](../../api/internal/api/analytics_handlers.go),
[`analytics.go`](../../api/internal/analytics/analytics.go), and
[`reportsAnalytics.ts`](../../web/src/lib/reportsAnalytics.ts).

## Trade summary

The server computes these fields in `GET /api/v1/analytics/summary` with
[`Summarize`](../../api/internal/analytics/analytics.go).

Let `N` be all included closed trades, `W` profitable trades, `L` losing trades,
and `B` breakeven trades. A trade is classified from its `net_pnl`: positive is a
win, negative is a loss, and exactly zero is breakeven.

| Field | Definition | Empty or undefined behavior |
|---|---|---|
| `total_trades` | `N = W + L + B` | `0` |
| `wins`, `losses`, `breakeven` | Counts by the sign of net P&L | `0` |
| `win_rate` | `W / N`; breakeven trades remain in the denominator | `0` when `N = 0` |
| `net_pnl` | Sum of net P&L | `0` |
| `gross_pnl` | Sum of each trade's before-fee P&L | `0` |
| `total_fees` | Sum of recorded trade fees | `0` |
| `gross_profit` | Sum of **positive net P&L** | `0` |
| `gross_loss` | Positive magnitude of all **negative net P&L** | `0` |
| `profit_factor` | `gross_profit / gross_loss` | `0` when there is no loss; the UI normally renders this as unavailable, not infinite |
| `avg_trade` | `net_pnl / N` | `0` when `N = 0` |
| `avg_win` | `gross_profit / W` | `0` when `W = 0` |
| `avg_loss` | `gross_loss / L`, reported as a positive magnitude | `0` when `L = 0` |
| `expectancy` | `(W/N × avg_win) − (L/N × avg_loss)` | `0` for an empty set |
| `largest_win` | Largest positive net P&L | `0` when there is no win |
| `largest_loss` | Largest loss magnitude | `0` when there is no loss |
| `median_trade` | Median net P&L across all closed trades | `0` when empty |
| `median_win` | Median positive net P&L | `0` when there is no win |
| `median_loss` | Median loss magnitude | `0` when there is no loss |
| `kelly_pct` | Full Kelly percent: `100 × (win_rate − (1 − win_rate) / payoff)`, where `payoff = avg_win / avg_loss` | `0` unless at least one win and one loss exist; negative values mean no positive estimated edge |
| `sqn` | `sqrt(N) × mean(net_pnl) / sample_stddev(net_pnl)` | `0` for fewer than two trades or zero dispersion |

`gross_profit`, `gross_loss`, profit factor, and expectancy are named using
standard trading terminology but currently operate on **net** trade outcomes.
Only `gross_pnl` is explicitly before fees.

## Equity, account value, and drawdown

These are two different datasets and must not be presented as synonyms.

### Realized equity curve

`GET /api/v1/analytics/equity-curve` merges cash transactions and closed-trade
net P&L in timestamp order. The curve begins at zero because an account's
starting balance is already represented by its opening-deposit cash transaction.

`equity = cumulative cash flows + cumulative realized net P&L`

`max_drawdown` is the largest positive currency difference `running_peak − equity`.
It is not a percentage. The curve contains no market value or unrealized P&L.

The Reports page derives percentage drawdown client-side from the same points:

`drawdown_pct = (equity − running_peak) / running_peak`

It is zero at a peak and negative below a peak. If the running peak is not
positive, it is reported as zero. `currentDrawdownPct` is the last value and
`maxDrawdownPct` is the most negative value. See
[`reportsAnalytics.ts`](../../web/src/lib/reportsAnalytics.ts).

### Reconstructed account value

`GET /api/v1/analytics/account-value` is a separate, Japanese-market
mark-to-market reconstruction implemented in
[`account_value.go`](../../api/internal/accountvalue/account_value.go).
It replays executions and the cash ledger across authoritative JPX sessions and
uses unadjusted daily closing prices.

| Field | Definition |
|---|---|
| `contributed_capital` | Cumulative external contributions and withdrawals according to the cash ledger |
| `cash_balance` | Replayed execution cash delta plus ledger cash |
| `open_position_value` | Sum of open quantities valued at the applicable daily close |
| `estimated_account_value` | `cash_balance + open_position_value` |
| `realized_pnl` | Realized result from position replay plus realized ledger items |
| `unrealized_pnl` | Open-position market value less replayed open cost |

Missing prices, invalid execution sequences, or unsupported corporate actions
produce warnings and an incomplete status. Value-dependent fields are `null`
rather than silently using zero. Portfolio points are summed only across
same-currency accounts; if any component is incomplete, the combined
value-dependent fields are also `null`.

## Daily P&L and breakdowns

`GET /api/v1/analytics/daily` sums net P&L by calendar day. `date_basis=open`
uses `opened_at`; all other values use `closed_at`. The selected timezone decides
the date key.

`GET /api/v1/analytics/breakdown` partitions the same closed-trade set and applies
the complete Trade summary formula to every group. Groups are ordered by net P&L
descending, then key ascending. Available dimensions are defined by the handler;
the notable time dimensions are:

- `session`: always uses `America/New_York`: Premarket 04:00–09:29, RTH
  09:30–15:59, Afterhours 16:00–19:59, and Overnight otherwise.
- `day_of_week` and `hour_of_day`: use the selected trader timezone.
- duration: `swing` crosses an Eastern Time calendar day; `scalp` closes on the
  same Eastern day with recorded duration under 600 seconds; other same-day
  trades are `day`.

See [`breakdown.go`](../../api/internal/analytics/breakdown.go),
[`session.go`](../../api/internal/analytics/session.go), and
[`duration.go`](../../api/internal/analytics/duration.go).

## R-multiple metrics

`GET /api/v1/analytics/r-summary` includes only closed trades with
`initial_risk > 0`; all others increment `excluded`.

`R = net_pnl / initial_risk`

Each R value is rounded to two decimals before aggregation. `avg_r`,
`avg_win_r`, `avg_loss_r`, `best_r`, and `worst_r` are arithmetic means or
extrema over included trades. `avg_loss_r` and `worst_r` stay negative. The
embedded summary uses the Trade summary formulas with R values as its P&L
input. Distribution buckets are `< -2R`, `[-2R, -1R)`, `[-1R, 0)`, `[0, 1R)`,
`[1R, 2R)`, and `>= 2R`.

Source: [`r_summary.go`](../../api/internal/analytics/r_summary.go).

## Web report-only metrics

These metrics are computed in the browser from already filtered trades or
equity points; they are not additional API fields.

| Metric | Definition |
|---|---|
| Rolling win rate | Wins divided by the configured trailing window size, emitted only after the window is full; breakeven trades occupy a window slot but are not wins |
| Metric evolution | Cumulative-to-date win rate, net P&L, profit factor, expectancy, and average P&L/trade at each day/week/month boundary |
| Median duration | Median recorded positive holding duration among the displayed scatter points |
| Average daily/weekly/monthly P&L | Total net P&L divided by the number of traded day/week/month buckets, not all calendar periods |
| Annualized P&L | `total net P&L × 365 / inclusive calendar-day span`; this is a currency run rate, not an investment return percentage |
| Average risk/trade | Mean positive `initial_risk`; missing and non-positive values are counted as excluded |

Weeks start Monday. Day grouping uses the market-day formatter supplied by the
Reports page. Source: [`reportsAnalytics.ts`](../../web/src/lib/reportsAnalytics.ts).

Home-page streaks, best/worst day, average holding times, and account
contributions are also client-derived from chronologically closed trades; see
[`homeInsights.ts`](../../web/src/lib/homeInsights.ts).

## Rule compliance

`GET /api/v1/analytics/compliance` scores closed trades by close date in the
selected timezone. If no supported rule is configured, or there are no trades,
the report contains no scored days.

- Risk violation: recorded `initial_risk` exceeds `max_risk_per_trade`.
  Missing risk increments `unknown_risk`; it does not automatically pass or fail.
- Daily-loss breach: running realized net P&L falls below
  `−max_daily_loss` at any point. A later recovery does not undo the breach.
- Trade-limit breach: the number of trades closed that day exceeds
  `max_trades_per_day`.
- Loss-streak breach: another trade closes after the configured consecutive-loss
  limit had already been reached that day.
- A day is compliant only when none of the configured breach conditions occurred.
  `compliant_pnl` and `breach_pnl` sum whole-day net P&L by that classification.

Source: [`compliance.go`](../../api/internal/analytics/compliance.go).

## Behavioral analytics

`GET /api/v1/analytics/behavior` reports insufficient data below 10 closed
trades but still returns the computed sections. Defaults are fixed in
[`behavior.go`](../../api/internal/analytics/behavior.go).

- Trade size is entry notional: `qty_opened × avg_entry_price`.
- Revenge trading examines a trade opened within 60 minutes of a losing close.
  It flags either a same-symbol re-entry within 15 minutes or size at least 1.5×
  the trailing median of up to 20 prior trades. At least five prior trades are
  required for that baseline; otherwise the losing trigger's size is used.
- Overconfidence flags the next trade after a streak of at least three wins when
  its size is at least 1.5× the streak's median size.
- Loss-aversion hold ratio is `average losing hold / average winning hold`.
  Holding time uses recorded duration, falling back to close minus open time.
- Give-backs are losing trades with recorded `MFE > 0`. `missed_profit` sums MFE
  for all such trades, while the response lists only the five largest MFE values.
  Losing trades without MFE increment `excluded`.

## Execution quality

`GET /api/v1/analytics/execution-score` reports five 0–100 axes. An axis is
`null`, not zero, when too few usable inputs exist: five for the overall report
and three for a weekly or monthly series bucket.

| Axis | Per-trade or aggregate definition |
|---|---|
| Entry | `100 × (1 − MAE / (MAE + MFE))`; requires recorded MAE and MFE with a positive combined move |
| Exit | `100 × clamp(net_pnl / MFE, 0, 1)`; only trades with `MFE > 0` are scored |
| Risk | With rules: 50% credit for recorded risk plus 50% for avoiding max-risk and daily-loss breaches. Without rules: 100 for recorded risk, otherwise 0 |
| Stability | Mean divided by sample standard deviation, preferably over R multiples and otherwise net P&L, mapped around 50 and clamped to 0–100; identical results are undefined |
| Tempo | 100 unless the trade is a same-symbol re-entry within 15 minutes or belongs to a sufficiently sampled day whose count is greater than 2× the trader's median daily count |

The composite is the weighted mean of defined axes: Entry 25%, Exit 25%, Risk
20%, Stability 20%, Tempo 10%. Missing axes are removed and the remaining
weights are renormalized. Values are rounded to one decimal place.

Source: [`execscore.go`](../../api/internal/analytics/execscore.go).

## Monte Carlo

`GET /api/v1/analytics/montecarlo` performs bootstrap sampling with replacement
from filtered per-trade net P&L. It assumes trades are independent and
identically distributed; it does not preserve streaks or serial correlation.

- Fewer than 10 trades returns `insufficient_data`.
- Default paths: 10,000; accepted values are clamped to 200–20,000.
- Default horizon: the sample size; maximum 1,000 trades.
- Terminal bands and fan bands are linearly interpolated empirical quantiles.
- `prob_negative` is the share of simulated terminal cumulative P&L below zero.
- Per-path maximum drawdown is the largest peak-to-trough currency loss.
- `risk_of_ruin` is the share of paths whose maximum drawdown reaches the ruin
  threshold. A zero/omitted threshold uses the historical sequence's maximum
  drawdown; if that threshold is also zero, risk of ruin remains zero.
- A supplied seed makes the result reproducible for the same sample and params.

Source: [`montecarlo.go`](../../api/internal/analytics/montecarlo.go).

## News prediction performance

News prediction performance is server-computed from persisted current evaluation
results; loading the report never calls a market-data provider.

`sample_count` contains only `validated` outcomes with a current supported rules
version and non-null directional correctness. Pending, incomplete, unavailable,
obsolete, and malformed results do not enter the denominator.

`hit_rate = 100 × correct / sample_count`

When `sample_count = 0`, `hit_rate` is `null`, not zero. Counts and rates are
grouped independently by source, horizon, asset, and news category. Date filters
refer to inclusive UTC news-publication dates.

Prediction return/direction thresholds, eligibility, market calendars, benchmark
behavior, and result-state semantics are documented separately in
[`prediction-validation.md`](prediction-validation.md). Aggregation source:
[`performance.go`](../../api/internal/predictionvalidation/performance.go).

## Change checklist

When adding or changing an analytics metric:

1. Update the authoritative server or client implementation and its focused test.
2. Define the sample, formula, unit, sign, timezone, rounding, and empty behavior.
3. Update this document and the OpenAPI schema when the API contract changes.
4. Keep realized equity and mark-to-market account value explicitly separate.
