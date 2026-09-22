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

## Detail

News titles link to `/news/{id}`. Direct navigation loads the current API record, with source
metadata, original text, summary and notes above the affected assets. Each asset contains its
own User/AI predictions; the existing editors are reused. Prediction details include timestamps,
catalysts, risks/invalidation and a horizon/status/result table ready for the later outcome data.
Missing/deleted records show a not-found state; API failures offer retry.

`e2e/news-detail.spec.ts` verifies direct routing, source links, hierarchy, edit/cancel/re-entry,
refresh read-back, prediction creation/deletion, loading/error/retry and missing/deleted records.
Set `E2E_SQLITE_PATH` only to a disposable test database to additionally seed and verify an AI
record alongside the User record; this fixture does not invoke an AI provider. All three News
browser suites passed together with Service Worker caching disabled.

Inspected screenshots: [detail](../screenshots/news-predictions/detail.png),
[AI record](../screenshots/news-predictions/detail-ai.png),
[deleted](../screenshots/news-predictions/deleted.png). Mobile remains outside validation scope.

## Structured AI news analysis

`coach.AnalyzeNews` reuses the Coach OpenAI-compatible transport/configuration and asks for
`news_analysis` JSON Schema before falling back to JSON-object mode on format rejection.
A single configured timeout bounds both attempts. The news prompt is separate from the
custom trade-coaching prompt; input news is treated as untrusted content.

The result contains a summary, category and up to 20 affected assets (stock/ETF/index), each
with symbol, market/exchange/name, direction, integer confidence 0–100, reasoning,
catalysts, risks and unique 1/3/5/10/20 trading-day horizons. Alphanumeric symbols such as
`285A` are preserved. Empty asset lists are valid. Required fields, unknown fields, types,
enums and ranges are checked locally in both response modes; malformed or partial output
fails as a whole. The service only returns tentative suggestions and never writes user data.

`go test ./internal/coach` covers schema/fallback, multiple assets, missing/null/invalid
fields, authentication/quota failures and timeouts with a local HTTP provider fixture.
These are deterministic transport tests, not validation of a live model's analysis quality.

## AI review in Web

From a news detail, **Analyze with AI** calls `POST /news/{id}/analyze` using the existing
AI Coach settings. This sends the title, original text, source, publication time and notes
to the configured provider. Analysis does not persist anything. Invalid provider output,
disabled settings or timeouts show a non-destructive error and retry.

Summary/category and each asset require explicit selection. Edit suggestions, reject rows,
exclude a row's prediction, or add a manual asset/prediction. Edited AI rows keep `ai` identity;
manual additions use `user`. Cancel discards the draft; reopening runs fresh analysis.
`POST /news/{id}/analysis/accept` validates selected items and writes them in one transaction.
Existing assets/predictions are never overwritten: accepted suggestions create separate records,
even for an existing symbol. Summary/category replace values only when selected and when the
review baseline still matches. Repeated acceptance is a new insertion, not a deduplicating sync.

`e2e/news-analysis.spec.ts` exercises the current Web and real API with a local HTTP AI fixture:
loading, success, selection in both directions, prediction inclusion in both directions,
editing, rejection, invalid horizons, manual additions, cancel/re-entry, refresh/API read-back,
provider failure/retry and empty suggestions. Existing User predictions remain unchanged.
No real model quality assessment or mobile validation was performed.
Screenshots: [review](../screenshots/news-analysis/review.png),
[accepted](../screenshots/news-analysis/accepted.png),
[provider failure](../screenshots/news-analysis/provider-error.png).

## Web Markdown export

News details expose **Export Markdown**. The list supports **Export selected (N)** and
**Export filtered**. Selection is limited to the current page and clears when changing
page, page size, filters, or leaving the list. Filtered export includes all matching pages.
The existing list filters run client-side, so export sends those exact result IDs to
`GET /news/export?id=...`, preserving symbol substring and display-timezone date semantics.
Single export uses `GET /news/{id}/export`; Markdown and filenames come from the server.

Empty selections/results disable export. Empty server documents, unavailable records,
permission errors, request failures, and detectable download failures show feedback.
Export requests bypass the Service Worker cache so stale files cannot hide API failures.
Downloads require a working API; the browser may still block or cancel saving after the
download is requested. Very large filtered sets remain subject to server/proxy URL limits
because the existing batch API uses GET. Mobile export UI is outside the current fork scope.
