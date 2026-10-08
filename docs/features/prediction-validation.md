# Prediction validation

## Using the performance report

In **News**, open **News performance** after saving predictions. It reads persisted
results; it does not fetch prices or trigger validation. Filter by source, symbol,
asset type, category, publication date or horizon, then use **Reset filters** to return
to the baseline. One prediction with 1D and 5D horizons contributes two samples.

![Actual report with synthetic pending and unavailable samples](../screenshots/showcase/prediction-performance.jpg)

The [showcase](../showcase-demo.md) produces six samples: four pending, two unavailable,
zero validated and zero incomplete. The two unavailable samples come from an explicitly
unsupported market; there is no hit rate with zero scored samples. Source=AI gives an
empty report because this seed has no AI predictions. That exclusion, reload read-back
and Reset were exercised in the production Web build against the disposable API.

To obtain scored outcomes, call the validation POST below with authenticated ownership
and a configured market-data provider. A prediction must be saved before the next regular
opening; each horizon becomes eligible only after its closing session plus 60 minutes.
News publication dates do not backdate predictions. Eligible daily bars need explicit
market/calendar coverage and sufficient adjustment/source-acquisition evidence.
Missing or unsupported evidence remains unavailable/incomplete, never a fabricated zero.

A fresh showcase cannot immediately create historical validated/incomplete examples through
supported APIs because save timestamps belong to the server. The synthetic September bars
are for valuation, not evidence for newly saved predictions. No database backdating, live
provider call or scored-outcome screenshot was used. Optional benchmark selection is API-only.
List/detail Pending placeholders are not a substitute for report/API evidence. Mobile not validated.

## Validation API and evidence rules

`POST /api/v1/news/{id}/predictions/{predictionId}/validate` evaluates and persists
each selected 1/3/5/10/20 trading-day horizon. No scheduler or Web UI is added.
`GET .../validations` returns retained evidence; `current=true` selects one evaluation
per selected horizon of the current prediction revision. Both endpoints require
ownership of the prediction and its parent news.

The engine implements `news-validation-v1` from the approved #59 specification:
next regular open strictly after the prediction save, inclusive trading-day counting,
close plus 60 minutes eligibility, unadjusted price returns and exact decimal ±0.001
neutral boundaries. Market must explicitly be JP/US and instrument must be stock.
Currency is fixed by the supported market (JPY/USD); cross-currency returns are not inferred.

Calendars are bounded, versioned XTKS/XNYS snapshots from exchange-calendars 4.13.2
covering 2020–2030, with official exchange spot checks documented alongside the data.
Out-of-range/unknown markets fail explicitly. Future exceptional closures require
updating the snapshot/version; source corrections preserve previous evaluation evidence.

Prices flow only through `marketdata.Service.RefreshBars`, which bypasses historical
cache entries to acquire post-close evidence. Legacy live providers record fetch start;
HTTP archives retain actual source acquisition timestamps. Every required session,
including the preceding corporate-action check day, must have valid daily evidence
acquired after the horizon's outcome close plus 60 minutes (refresh the full window when an archive contains older acquisition times). Missing provenance, adjustment
uncertainty, duplicates, missing sessions and corporate-action candidates cannot score
as validated. No source-name-specific validation rules exist.

Audit JSON stores the prediction/asset snapshot and revision hash, rule/calendar versions,
open/close UTC sessions, policies, separate resolutions and horizon, request and bar evidence,
source acquisition time, corporate-action candidates/limitations and null-or-valid results.
Source errors remain unavailable; absent data never becomes zero return. Listing,
delisting or suspension causes are not invented without authoritative metadata.
Corporate-action detection is the existing split-ratio/OHLC heuristic, not exhaustive event coverage.

Edits increment a persistent prediction revision. Original content is never rewritten by
validation. Identical evidence updates attempt time only; changed evidence is retained
as a separate evaluation and may retract a previously validated result. `current` reflects
the newest attempt, not the best outcome. Parent deletion cascades to evidence. Both
SQLite and Postgres schemas/queries support this contract.

## Optional benchmark

The validation POST optionally accepts:

```json
{"benchmark":{"symbol":"1306","market":"JP","currency":"JPY"}}
```

The symbol identifies a security resolvable through the shared stock OHLCV contract
(for example a selected proxy ETF); no index, proxy or currency is guessed. Each call
explicitly supplies the desired association. An omitted/null benchmark clears it for
that evaluation and yields `benchmark_status=not_requested` with null benchmark/excess
values. Earlier evidence and selections remain in history.

The benchmark must have the same currency and identical dates, regular opening UTC
instants and closing UTC instants on **every** session of the asset window. A holiday,
extra session, timezone or early-close difference is not repaired by intersecting dates
or carrying prices forward. Currency/calendar failures are explicit independent benchmark
states. Matching windows are evaluated by the same engine and shared market service,
including source-time, missing-bar, adjustment and corporate-action checks.

`benchmark_return = BH/B0 - 1`; `excess_return = asset_return - benchmark_return`.
Only two validated sides yield excess and excess-direction correctness; classification
uses exact decimal rationals with the same ±0.001 neutral band. Primary direction
correctness always uses raw asset return. Asset +2% and benchmark +3% yield -1 percentage
point excess: a bullish prediction is correct on raw return and incorrect on excess.
Benchmark failure never clears a valid asset return. A validated benchmark can be retained
when the asset is unavailable, with null excess.

Benchmark identity, independent evaluation evidence, returns and states use the existing
immutable audit snapshot persistence. A changed selection/evidence creates an evaluation;
identical evidence updates attempt time. No duplicate provider, cache, schema migration,
Web UI or automatic benchmark selection is added. Mobile was not validated.

## Performance aggregation

`GET /api/v1/news/performance` reads persisted results without fetching prices or running
validation. Optional filters: `source=ai|user`, exact `symbol` (case normalized),
`asset_type=stock|etf|index`, exact `category`, `horizon=1|3|5|10|20`, and inclusive
`from`/`to` news-publication dates (`YYYY-MM-DD`, UTC). Filters combine with AND; invalid
values return 400. All reads are owner-scoped.

The unit is **one current prediction × one selected trading-day horizon**, not one news
article or one independent trade. Total equals pending + validated + unavailable + incomplete.
Each horizon contributes once using the latest current-revision evaluation from the existing
validation history rules. Missing current results are pending, including after prediction edits.
A newer failure replaces an older successful result; retained audit history is not extra samples.
Old rules or malformed finalized results without correctness are unavailable.

Counts are combined at the top level. Rates remain separated by `source` in `by_source`,
`by_horizon`, `by_asset` (type/market/exchange/symbol), and `by_category`.
`sample_count` is the number of validated outcomes; `correct` is its directional-hit numerator;
`hit_rate = 100 * correct / sample_count`. Pending, unavailable and incomplete never enter that
denominator. Zero samples yield `hit_rate: null`, never zero percent. Small samples retain their
actual counts; no confidence or significance is implied. Rates use raw directional correctness,
not optional benchmark excess returns. Empty breakdowns are `[]`.

The implementation reuses per-prediction history reads inside a transaction. This suits a
personal journal; batching is the upgrade path if measured volume makes it slow.
`go test ./internal/predictionvalidation ./internal/api` covers reproducibility, mixed statuses,
AI/User separation, one/zero-sample rates, filtering, isolation, retractions and stale revisions.

## Web performance report

Open **News → Performance report** (`/news/performance`). The report reuses `Page`, `Card`,
`StatCard`, `DataTable`, `Field` and native form controls from the existing dashboard/report UI.
It shows all five status counts plus AI/User, horizon, asset and category breakdowns from the
aggregate API. The browser formats rates but never computes results or denominators. Every rate
shows `n` and the number correct; pending-only groups show **Not available · n=0**.

Source, asset type, symbol, category, trading-day horizon and UTC publication-date filters combine
on the server. Filters are stored in the URL for refresh/back re-entry. **Reset filters** restores
the baseline. Loading, empty and failed requests have explicit states; failed requests offer retry.

`e2e/news-performance.spec.ts` passed against the working-tree API/Web with a fresh disposable
SQLite database. It seeds known September 2026 prediction times, then calls the real validation
endpoint against a local Generic Bars HTTP fixture to produce hits, misses, pending and unavailable
outcomes. It verifies all filters/exclusions/reset, URL read-back, refresh/re-entry, zero denominators,
loading, empty, error/retry, and the News entry link. The report makes no additional market calls.
Set `E2E_SQLITE_PATH` only to that disposable database; the fixture adjusts timestamps/source there.
The test API must use `TM_MARKET_DATA_PROVIDER=http` and
`TM_MARKET_DATA_HTTP_BASE_URL=http://127.0.0.1:18963`. Live market quality and mobile were not tested.

Inspected screenshots: [overview](../screenshots/news-performance/overview.png),
[pending-only](../screenshots/news-performance/pending.png),
[empty](../screenshots/news-performance/empty.png), [error](../screenshots/news-performance/error.png).
