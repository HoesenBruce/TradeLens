# Mixed-currency Summary contract (#175)

`GET /api/v1/analytics/summary?target_currency=JPY` accepts mixed account scopes.
The request target is returned as `target_currency`; `currency` is the actual
output currency. Without a target, a same-currency scope keeps its native
currency, an empty account scope returns an empty currency, and a mixed scope
returns `400 mixed_currencies`.

Each closed trade's persisted `pnl_currency` supplies the source currency for
net P&L, gross P&L (including the legacy net+fees fallback), and total fees.
Inputs are converted once, before `analytics.Summarize` recomputes every statistic.
Stored trades, execution grouping, broker accounting and replay are not modified.

## FX timing and failure

`fx_policy: latest` uses the existing market service's latest rate and 15-minute
cache. `fx_rates` records each non-identity source/target rate, timestamp and
provider. These are current-value comparisons, not historical trade-date FX or
broker-settled returns. Identity conversions need no network request.

Unknown account/trade currencies fail with `unknown_currency`. Missing, failed,
non-positive or non-finite FX fails with `502 fx_unavailable`; no partial summary
is returned. Invalid target syntax returns `400 bad_request`.

Web mixed scopes explicitly request the selected display currency. With no
selection (Auto), their defined target is USD. Same-currency scopes keep the
native API contract and existing display conversion. Query keys include the
target so switching display or account scope cannot reuse the wrong result.
Response currency remains the monetary source of truth. Mixed Header balance
and funding return, Reports deposit-percentage mode and annual-goal progress
are unavailable until their native cash/goal inputs can be normalized.

## Other consumers

The endpoint audit and remaining normalization work are tracked in
[#179](https://github.com/HoesenBruce/TradeLens/issues/179).
Equity and Daily normalization is delivered by #181 (part 1 of #179); see
`../issue179/README.md` for the new contract and validation. Breakdown/Reports/Playbook,
trade rows/account contribution and Wrapped retain their mixed-currency rejection. Explicit portfolio
selection still disallows incompatible currencies; All Accounts supports mixed
Summary. Historical account-value reconstruction/combining remains unchanged.

## Validation

- API tests compare every Summary field against normalized JPY+USD samples for
  JPY and USD targets, with unequal win/loss counts and fees; inverse cache,
  identity, account narrowing, unknown IDs, invalid targets and failed FX covered.
- Web hook tests cover JPY -> USD -> JPY, single -> All Accounts, Auto USD,
  response-currency formatting without a second FX request, and failed Summary.
- Full Go suite passed, including existing historical account-value tests.
- Final full Web suite on the latest main base: 179 files / 1025 tests passed;
  affected-path tests: 6 files / 77 passed. Web check: 0 errors (74 existing
  warnings). Go vet/build and Web production build passed.
- Browser fixture: isolated API on 8095 and Web on 5185; JPY trades +200/-110
  (gross -100, fees 10), USD trade +20, September 2026. Live latest USD/JPY
  157.0919952392578 produced JPY 3231.84 / USD 20.57; rendered JPY rounded to
  3232, fees 10, average win 1671, average loss 110, win rate 67%, PF 30.38.
- Browser happy/inverse/read-back/re-entry/cancel/reset: All -> JPY -> All ->
  USD -> All; JPY -> Auto USD -> JPY; reopened menus kept their checkmarks,
  Escape left values unchanged, reload preserved account/display selections.
- Reports switched Mean -> Median -> Mean. Median trade JPY 200, largest win
  JPY 3142, largest loss JPY 110 and SQN 1.04 matched normalized API samples.
  Switching Reports back to USD produced net 20.57 and expectancy 6.86.
- At 390px, the Web More drawer changed the target to JPY and the summary
  retained correct amounts with no visible overflow.
- A disposable ZZZ-currency trade forced real FX failure: API 502, Header “—”,
  Summary error and no partial amounts; switching to JPY recovered net 90.
  Selecting an empty JPY account showed “No trades” rather than stale statistics.
- Final funding guard: with a seeded annual target of 10000, mixed scopes did
  not publish goal progress. Reports % -> $ toggled safely; without normalized
  deposits the existing display context fell back to absolute monetary amounts.
- Screenshots: [desktop](desktop-jpy.png), [390px](narrow-jpy.png),
  [Reports median/SQN](reports-median.png), [FX failure](fx-failure.png).
- Existing Reports Performance “Gross” presentation uses the sum of win/loss
  buckets, while API `gross_pnl` is before-fees P&L. This inherited presentation
  difference was observed during acceptance and is preserved per the accounting
  guardrail; it is not a second FX conversion.
- Mobile apps not validated; outside current fork scope. A narrow Web viewport
  is Web responsive verification, not iOS/Android validation.
