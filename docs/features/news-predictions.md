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

## Manual editing

The News page supports creating, editing and deleting User predictions. AI records are visible
but cannot be edited or deleted through the manual API. The affected asset cannot be changed
on an existing prediction. Saving replaces the selected horizons atomically; at least one is
required. GET `/api/v1/news` and `/api/v1/news/{id}` include predictions and their horizons.

Browser validation uses a disposable API database and the current Web working tree:

```
cd web
E2E_API_URL=http://localhost:8092/api/v1 E2E_WEB_URL=http://localhost:5175 \
E2E_EMAIL=<test-user> E2E_PASSWORD=<test-password> pnpm run e2e e2e/news-predictions.spec.ts
```

Verified create/edit/delete, refresh read-back, cancel/re-entry, invalid confidence, empty horizons,
select/unselect horizons, nullable confidence and API read-back. Rendered screenshots:
[editor](../screenshots/news-predictions/editor.png), [saved](../screenshots/news-predictions/saved.png).
Mobile not validated; outside current fork scope. Outcomes remain pending; no AI calls or
validation calculations are implemented here.

## Overview

The News overview shows each thesis's published date, source, affected assets, predictions,
confidence and trading-day horizons. Open **Predictions** to use the manual editor.
Filters combine direction, symbol, published-date range and validation status; dates use the
same display timezone as the table. News filters persist separately from trading filters,
and **Reset filters** restores the unfiltered list. Pagination reuses the existing component.

There are no outcome records yet, so all theses explicitly show **Pending validation** and
**Validated** correctly produces an empty result. Actual result calculation remains in #47.
Filtering/pagination operate on the existing full News response.

`e2e/news-list.spec.ts` exercises real API records on both sides of date/direction filters,
exclusion, reset, refresh and route re-entry, forward/back pagination and page size. Network
interception is used only to verify loading and error/retry states; the main assertions use
real API data. Screenshots: [overview](../screenshots/news-predictions/overview.png),
[filtered](../screenshots/news-predictions/filtered.png).
