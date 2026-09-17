# Japanese Daily Trade Chart — Design

**Issue:** #13
**Status:** Approved for implementation

## Reused path

The existing `marketdata.Service` already provides provider-agnostic bar loading,
single-flight protection, memory/SQLite caching, and `/market/bars`. The existing
`TradeChart` renders candlesticks and execution-direction markers. This work extends
that path; it does not add a second provider, cache, endpoint, or chart.

## Scope

- A four-digit stock symbol is a Tokyo Stock Exchange equity: Yahoo requests it as
  `<code>.T`, uses `Asia/Tokyo`, and permits the Web chart fetch.
- The existing response explicitly states instrument, source, timezone, and that
  Yahoo's returned `close` is unadjusted. Daily bars also carry `market_date`.
- The chart uses the response timezone. Each execution keeps the existing directional
  arrow and gains a price line at its actual execution price.
- `trade_type` is additive execution metadata derived from persisted SBI lot data;
  unknown brokers retain their side as the type.

## Deliberate limits

- This issue does not add intraday Japanese bars, a new provider, corporate-action
  adjustment, account reconstruction, or a database migration.
- Multiple same-day executions remain individual API fills and individual price lines;
  the detail page's fills table is their inspectable list.
