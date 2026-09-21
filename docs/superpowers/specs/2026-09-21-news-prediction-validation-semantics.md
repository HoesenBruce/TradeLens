# News prediction validation semantics

**Issue:** #59 · **Parent:** #47 · **Implementation:** #60 (raw returns), #61 (optional benchmark)

**Semantics version:** `news-validation-v1`

**Status:** This specification defines the contract for subsequent implementation. Outcome
calculation and persistence are not implemented yet.

Validation records price behavior after a prediction is made. It does not establish that the
news caused the move, represent executable returns, or constitute an investment backtest.
This PR adds only a specification; it changes no API, database, or Web behavior. Mobile is
outside the acceptance scope.

## 1. Existing capabilities and implementation boundaries

- Reuse `api/internal/marketdata.Service.GetBars`, `Request`, `Response`, the Yahoo/Finnhub
  providers, and existing memory/database caches. Request `Interval = "D"`; do not introduce
  a news-specific downloader or cache.
- Reuse `MarketTimezone`, `Bar.MarketDate`, and Japanese symbol mapping. For example, request
  `285A` as a stock; Yahoo maps it internally to `285A.T`. An arbitrary symbol string in a
  news item is not sufficient evidence of its market.
- Reuse `FindCorporateActionCandidates` / `CorporateActionWarnings`. Candidates are not
  confirmed corporate actions.
- `predictions` currently stores source, direction, confidence, created_at, and updated_at;
  `prediction_horizons` stores 1/3/5/10/20. Outcomes, prediction revision snapshots, and trading
  calendars are not implemented yet.
- The shared service currently returns `adjustment_status = unadjusted`. Yahoo's
  `applyYahooSplits` attempts to restore prices using split events within the request.
  The older corporate-action specification's `split_adjusted` description is not the current
  contract. The label alone proves neither complete event coverage nor that historical OHLC
  values are comparable across corporate actions.
- #60 must identify and version the JP/US exchange calendar source and use it at the shared
  market-data boundary. Existing timezone/date normalization is not a trading calendar:
  neither dates with bars nor Monday through Friday can substitute for one. Fail explicitly
  when the source is unavailable. This specification selects no new library and does not
  require #59 to implement calendars, scheduling, backfills, AI calls, or an outcome UI.

## 2. Prediction time, reference price, and trading-day counting

Support only JP/US stocks with an identified trading market and currency. Other instruments
or assets with an unknown market return `unavailable / unsupported_instrument`; do not infer
US market membership from the default New York timezone.

`prediction_as_of` is the actual save time of the current prediction revision in UTC, not the
news publication time. Initially use created_at. Every subsequent prediction edit creates a
new revision and as_of; old outcomes do not transfer to the new revision. Publication time is
context only: entering historical news later must not grant a reference price before entry.
Existing updated_at can serve as migration snapshot evidence, but cannot alone identify a
unique revision. #60 must persist an immutable content snapshot or content hash with an
explicit revision identifier. The validator must not rewrite the original news, prediction
source, body, or prediction content.

v1 uses P0, the **next regular-session opening price strictly after prediction_as_of**.
This consistent daily-bar convention deliberately excludes the immediate response between
intraday news and that day's close, and does not use a past closing price.

| Prediction save time (asset market local time) | Reference date D0 / P0 |
| --- | --- |
| Before the open on a trading day | Same day / regular daily-bar Open |
| Exactly at the open, during the session, or during the lunch break | Next trading day / its daily-bar Open |
| Exactly at the close, after the close, or during extended-hours trading | Next regular open strictly after save time / corresponding daily-bar Open |
| Weekend or exchange holiday | Next trading day / its daily-bar Open |

A prediction saved exactly at the open also moves to the next trading day because second-level
timestamps cannot establish ordering within that instant. A suspension does not move D0:
if its valid daily bar is missing, do not defer to resumption or fill with the previous close.

For H ∈ {1, 3, 5, 10, 20}, **D0 counts as trading day 1**. DH is the Hth trading day starting
at D0; PH is its regular daily-bar Close. Thus 1D runs from D0's open to D0's close, and 3D
runs from D0's open to the third trading day's close. Count exchange trading days, not calendar
days or returned bars. An individual stock's suspension still counts as a trading day.
Calculate and track each horizon independently; shorter horizons need not wait for the longest.

## 3. Market dates, completeness, and availability

- Use `Asia/Tokyo` for JP and `America/New_York` for US. Store instants in UTC and trading
  dates as exchange-local `MarketDate`. Use IANA timezone rules for US daylight saving time,
  not a fixed UTC offset.
- Opens, closes, holidays, exceptional closures, and early closes come from the versioned
  market calendar. A lunch break does not create another trading day. Timestamps must carry
  timezone information; missing/invalid timestamps yield `unavailable / invalid_prediction_time`.
- Do not treat daily `Bar.Time` as the closing instant. The calendar defines DH's close.
  Before that close, a provider's current daily bar cannot supply a validation Close.
- v1 sets `eligible_at` to 60 minutes after the close. This is a product waiting rule, not a
  provider completeness guarantee. Before it, the result is pending; afterward, missing or
  incomplete data produces the failure states below and may be retried. Use a market-data
  snapshot fetched after eligible_at; a cache-hit time must not masquerade as an upstream
  fetch time. Current Response does not expose fetched_at. #60 must supply shared provenance
  or refresh through the shared service, without bypassing it to download prices.
- Require exactly one valid daily bar for every expected trading day from D0 through DH.
  Weekends/holidays require no bars. Prices must be finite, positive, and satisfy valid OHLC
  relationships. Zero volume alone neither proves suspension nor permits fabricated prices.
  Conflicting duplicate dates, wrong market dates, non-trading-day bars, and known intraday
  snapshots cannot constitute complete input.
- A missing intermediate bar also makes the result incomplete, even if both endpoint prices
  exist. Do not interpolate, forward-fill, skip missing days, or silently extend the window.
  Distinguish wholly absent data from partial data. A network failure does not mean no price move.

## 4. Adjustment policy and corporate actions

v1 calculates **unadjusted price returns** within one currency, provider, and price convention.
It excludes dividend reinvestment, fees, slippage, taxes, and FX returns, and must not be labeled
total return. Do not mix adjusted close with unadjusted open. Unknown/non-unadjusted
`adjustment_status`, inconsistent endpoints, or stitched sources make the result incomplete.

For corporate-action checks, additionally request the trading day before D0 to detect boundary
anomalies at D0; it is not part of the return horizon. If this check bar is missing, return
`incomplete / corporate_action_evidence_missing`.
An incomparable event or unresolved candidate within D0…DH, inclusive, such as a split, reverse
split, rights issue, or merger, conservatively yields `incomplete / corporate_action_boundary`.
Do not score direction or automatically alter prices/quantities. This applies even to an event
on D0 when both endpoints appear comparable. Explicitly rejected false-positive candidates do
not block validation, but their audit evidence must remain available.

Record known cash dividends as warnings; do not correct an ex-dividend gap as a split. Price
returns still exclude dividends. An unexplained anomaly flagged by existing candidate detection
remains incomplete; do not assume it is a dividend. No detected candidates does not establish
complete corporate-action verification. Persist the detection method and coverage limitations
without claiming authoritative event coverage. Economic returns across splits require a later
version; #60/#61 must not implicitly introduce an adjustment model.

## 5. Returns and directional correctness

Persist returns as fractions; multiply by 100% for display. Only validated results contribute
to accuracy statistics. Missing values are null, not zero.

```
asset_return = PH / P0 - 1
benchmark_return = BH / B0 - 1
excess_return = asset_return - benchmark_return
neutral_epsilon = 0.001  # 10 bps, or 0.10%; fixed in v1
```

Compare returns before display rounding. Zero and both threshold boundaries are neutral.
Do not weight by confidence or vary the threshold by volatility/horizon. Prevent binary
floating-point error from moving a mathematically exact ±0.001 outside the neutral interval.
Lock this behavior with boundary and immediately adjacent tests; comparing decimal prices
PH against P0 × (1 ± epsilon) is one possible implementation.

| Observed return r | Observed direction | Bullish correct | Bearish correct | Neutral correct |
| --- | --- | --- | --- | --- |
| r > 0.001 | bullish | Yes | No | No |
| r < -0.001 | bearish | No | Yes | No |
| -0.001 ≤ r ≤ 0.001 | neutral | No | No | Yes |

The primary `direction_correct` always uses asset_return. Optional `excess_direction_correct`
applies the same rule to excess_return and never replaces the primary result. For example,
asset +2% and benchmark +3% make a bullish prediction correct on raw direction and incorrect
on excess direction. Observed direction and correctness are null unless validated.

## 6. Optional benchmark and window alignment (#61)

Explicitly select a benchmark security, market, and currency resolvable by the shared service.
Do not guess an index or proxy ETF. When none is selected, `benchmark_status = not_requested`,
all benchmark values are null, and the raw result is unaffected.

v1 requires the same currency and identical expected trading dates and regular opening/closing
UTC instants throughout D0…DH. B0 therefore uses the asset's D0 open, and BH uses its DH close;
the benchmark must not count a separate H-day window. Compare both calendars across the entire
window. Matching date labels with different timezone/market sessions are still misaligned.
Non-overlapping holidays, additional benchmark trading days, or different early closes yield
`unavailable / benchmark_calendar_mismatch`. Different currencies yield
`unavailable / benchmark_currency_mismatch`. Do not intersect calendars, choose nearby dates,
carry forward prices, or convert currencies. Asynchronous cross-market benchmarks require a
later semantics version.

Apply the same fetch-time, missing-bar, adjustment, and corporate-action rules to the benchmark.
Benchmark failure affects only benchmark_status and excess fields; retain a validated raw asset
return. Publish excess only when both sides are validated. If the asset is not validated, the
benchmark's own result may be stored, but excess and its directional correctness remain null.

## 7. States and reruns

Each prediction revision × horizon has an asset state. The benchmark has an independent state
with the additional value not_requested. Evaluate the following table in order, persisting
reason_code and an explanation rather than collapsing failures into permanent pending.

| State | Conditions and examples | Publishable result |
| --- | --- | --- |
| unavailable | Unsupported input/no reliable calendar, or fetch failure/no valid data after eligibility | Null result; e.g. calendar_unavailable, provider_error, no_bars |
| pending | Input and calendar resolve, but eligible_at has not been reached | Known scheduled dates; null return/correctness |
| incomplete | Eligible, with partial data but missing days, conflicts, invalid prices, insufficient fetch-time evidence, or adjustment/corporate-action uncertainty | Evidence and missing dates; null official return/correctness |
| validated | Eligible and all completeness checks pass | Reference price, outcome price, return, observed direction, correctness |

Prices need not be requested before eligibility; provider_error/no_bars arise only during an
eligible evaluation. If the calendar cannot resolve, DH/eligible_at are null; do not guess dates.
Reevaluate each retry under the same version. Pending/unavailable/incomplete can all become
validated. No fixed retry count automatically turns an unavailable result into a wrong prediction.

Identical input revision, rules version, calendar version, and price snapshot must reproduce
the same result without double-counting statistics. With identical evidence, update attempt
metadata only. Provider history corrections or calendar updates create a new evaluation revision
and preserve earlier evidence/results. New evidence may downgrade validated to incomplete or
unavailable; do not retain only the best outcome. After a prediction edit, old results belong
only to the old revision, and the current view reflects the new revision's state. Adding/removing
horizons does not rewrite historical evidence. This does not require retaining deleted news
forever: preserve existing owner isolation and parent-record cascade deletion policies.

## 8. Required persisted audit fields

These are semantic requirements; #60/#61 choose the minimal schema. Do not store only
percentages or cache keys. Market-data caches can expire, so persist result evidence alongside
the result; this is not a second market-data service cache.

| Group | Required fields |
| --- | --- |
| Identity | Owner/news/asset/prediction IDs, prediction revision ID/content hash and snapshot, source, direction, confidence, horizon H |
| Time and rules | prediction_as_of, nullable news published_at, rules version, epsilon, calculation time, eligible_at, calendar source/version, timezone, market, currency |
| Window | D0, DH, reference/outcome fields open/close, corresponding trading instants in UTC, expected trading dates, missing dates |
| Price evidence | Normalized symbol, provider symbol/instrument, interval, provider/source, adjustment_status, request window, actual fetched_at, cache flag, daily-bar snapshots/hashes used |
| Corporate actions | Check window including the preceding day, candidates/events, status, source, warnings, detection and coverage limitations |
| Asset result | P0, PH, asset_return, observed_direction, direction_correct, status, reason_code, explanation |
| Benchmark result | Nullable benchmark identity, independent window/source/price/snapshot/corporate-action evidence, benchmark_status/reason, B0, BH, benchmark_return, excess_return, excess_direction_correct |
| Reruns | Evaluation revision ID, previous evaluation reference, input/evidence fingerprint, last attempt time; distinguish refetching from recomputation of the same snapshot |

Preserve existing price precision; do not round returns to display precision before persistence.
Unknown audit values are null with an explanation. Do not invent fetched_at, completeness, or
provider guarantees. Missing Response provenance must be supplied at the shared market-data
boundary. Source snapshots must support recomputation independently of an expiring cache.

## 9. Deterministic acceptance examples for implementation

#60/#61 must turn these inputs/expectations into tests. They are not evidence that this PR has
exercised a validation engine. Time examples use explicit test calendars, with no network
assumption about closed days; actual calendar integration requires separate source verification.

1. Test JP calendar: Monday open, Tuesday holiday, Wednesday/Thursday open. A prediction saved
   before Monday's open has D0 Monday, 1D Monday close, and 3D Thursday close. Saving during
   Monday's session/lunch break/at the close starts at Wednesday's open. Weekend entry starts
   Monday. Entry exactly at the open uses the next trading day. Suspensions/missing bars do
   not move these dates.
2. For trading days S1…S20 and D0=S1, 1/3/5/10/20D use S1/S3/S5/S10/S20 Close respectively.
   S1's 1D can be validated while S20 remains pending before its eligibility time.
3. Across US daylight saving transitions, select D0 by New York local opening time while the
   corresponding UTC time changes. On an early-close day, eligible_at is actual close plus
   60 minutes. Interpret JP dates in Tokyo time, not by slicing a UTC date.
4. `285A` uses existing Japanese symbol mapping. Unknown markets/non-stock instruments do not
   default to US, and invalid timestamps fail explicitly.
5. With P0=100, PH=101/99/100 yields bullish/bearish/neutral. Both 100.1 and 99.9 are neutral;
   100.1001 is bullish and 99.8999 bearish. Verify correctness for all three predicted directions.
6. An intraday bar returned before eligibility remains pending. After eligibility, an old
   intraday cache alone is incomplete. No bars yields unavailable; missing intermediate days,
   suspension, conflicting duplicates, or NaN/zero prices yield incomplete. Completing evidence
   allows validated with the original D0/DH. No data must never yield a 0% return.
7. A two-for-one split moving unadjusted prices from 100 to 50 produces a corporate-action
   boundary and incomplete, not a correct bearish prediction. Events on D0 also block.
   Missing preceding-day evidence or unknown adjustment status blocks validation. An explicitly
   rejected candidate permits reevaluation.
8. No benchmark leaves the asset unaffected. Asset +2%, benchmark +3% yields excess of -1
   percentage point. Missing benchmark prices/corporate actions do not destroy the asset
   result. Currency or window/calendar mismatches must not silently change calculation dates.
9. Repeating a snapshot does not duplicate statistical contributions. Historical price revisions
   can retract validated while preserving old evaluations. Editing creates a new as_of/revision.
   Original news, User/AI source, and other owners' records remain unchanged.

## 10. Delivery boundary

#59 delivers this specification for review in one documentation PR. #60 implements asset
validation, audit persistence, and the relevant tests above; #61 adds benchmark validation.
Shared-interface gaps belong to subsequent implementation and are not claimed as resolved here.
Changes to reference prices, trading-day counting, thresholds, adjustments, or alignment must
increment the semantics version rather than silently rewrite old outcomes. When Web surfaces
show actual validation results, exercise them end to end against a real API under repository rules.
