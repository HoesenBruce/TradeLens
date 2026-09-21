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

Mobile was not validated; no UI or mobile contract was changed. Benchmark evaluation is #61.
