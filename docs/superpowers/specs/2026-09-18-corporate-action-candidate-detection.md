# Corporate-action candidate detection

**Issue:** #14

`marketdata.Service.DetectCorporateActions` reuses the existing provider, memory
cache, and database cache. It requests daily bars; there is no separate historical-
price loader and no mutation of imported executions. Yahoo OHLC history is marked
`split_adjusted`, so its split-event metadata is retained on the effective day's bar
instead of pretending an adjusted series is unadjusted.

Authenticated clients can inspect the same structured results through
`GET /api/v1/market/corporate-actions?symbol=...&from=...&to=...`.

The detector first uses a provider-reported split ratio when present. Otherwise it
reports an unconfirmed candidate when the previous close and next open differ by at
least 1.5x. A ratio near 2:1, 3:1, 4:1, 5:1, 6:1, or 10:1 plus consistent OHLC
scaling is classified as a split or reverse split. Other large moves remain
`other_corporate_action`; unexplained jumps in adjusted input are explicitly
`adjustment_mismatch`. Every result includes the instrument, effective date,
suspected ratio, confidence, evidence, source, adjustment status, and status.

These are heuristics, not authoritative corporate-action records. False positives
and false negatives are expected: genuine price gaps may be candidates, while
unusual ratios or noisy bars may not look like a split. All inferred results stay
`unconfirmed` until a future authoritative source or review workflow changes that
status.

`CorporateActionWarnings` lets account reconstruction mark a date range incomplete
or unsupported when it contains an unconfirmed boundary. It does not adjust
position quantities, prices, cash, or broker history. The reconstruction work in
#30 should put these warnings in its public response when that feature is built.
