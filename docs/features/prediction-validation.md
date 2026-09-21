# Prediction validation

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
