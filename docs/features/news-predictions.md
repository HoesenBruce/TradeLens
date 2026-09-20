# News predictions

A prediction belongs to one affected `news_assets` record, which links it to its news thesis
and owner. Deleting the asset or news cascades to predictions and their horizons.

- `source`: `user` or `ai`; separate records can coexist on the same asset. Updates cannot
  change source or asset. Update/delete queries require the expected source and owner.
- `direction`: `bullish`, `bearish`, or `neutral`.
- `confidence`: optional integer percentage, 0–100 inclusive; null means unspecified.
- `reasoning`, `catalysts`, `risks`, `invalidation`: text captured with the prediction.
- `created_at` and `updated_at`: database timestamps.
- `prediction_horizons`: unique trading-day counts of 1, 3, 5, 10, or 20 per prediction.
  These are trading-day horizons, not calendar-day deadlines. Store callers replace horizons
  and update the prediction together using `store.InTx`.

SQLite and Postgres enforce the same constraints. All store queries are owner-scoped.
This persistence layer does not call AI or compute outcomes.

Run the shared store checks with `cd api && go test ./internal/store -run TestPredictions`.
Set `TM_TEST_DATABASE_URL` to a disposable Postgres database to exercise the Postgres case too.
