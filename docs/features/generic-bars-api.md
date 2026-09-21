# Generic Bars API v1

Select `TM_MARKET_DATA_PROVIDER=http`, set `TM_MARKET_DATA_HTTP_BASE_URL` to an
HTTP(S) service origin (optional path prefix), and optionally set
`TM_MARKET_DATA_HTTP_API_KEY`. Authentication uses only `Authorization: Bearer <key>`.
The legacy Yahoo/Finnhub configuration remains unchanged.

`GET /v1/bars` accepts `symbol`, `instrument_type`, `interval`, `from`, `to`.
Symbols use the existing mapping (`285A` → `285A.T`, `5803` → `5803.T`, `AAPL` unchanged).
Intervals use shared codes (`1`, `5`, `15`, `30`, `60`, `240`, `D`, `W`, `M`);
the adapter passes any nonempty code through for the server to accept or reject.
Both times are RFC3339 UTC instants; `from` is inclusive and `to` exclusive.

```json
{
  "symbol": "285A.T",
  "interval": "D",
  "timezone": "Asia/Tokyo",
  "source": "archive",
  "adjustment_status": "unadjusted",
  "bars": [{
    "timestamp": "2026-09-18T00:00:00Z",
    "market_date": "2026-09-18",
    "open": 1000, "high": 1010, "low": 995, "close": 1005, "volume": 123400
  }]
}
```

`bars` is required; no local data is HTTP 200 with `bars: []`. Each bar requires an
offset-aware RFC3339 timestamp and all OHLCV fields. Optional `split_ratio` means
new shares per old share. Prices must be positive with valid OHLC relationships;
volume is nonnegative. Duplicate/out-of-range/partial/invalid records reject the
whole response. Results are sorted by timestamp. Market date is derived from the
supplied IANA timezone; a supplied conflicting date is rejected. Missing source or
timezone remains unknown; missing adjustment status becomes `unknown` (other values:
`unadjusted`, `adjusted`). No unavailable corporate-action metadata is fabricated.
Symbol and interval must echo the request exactly.

Errors use non-2xx status (400 invalid input/range, 401 authentication, 422 unsupported
interval, 413 range too large, 503 storage unavailable) and may include
`{"error":{"code":"unsupported_interval","message":"Unsupported interval"}}`.
Clients report only the status; remote bodies/transport errors are never logged by
this adapter. The timeout is 20 seconds, body limit 16 MiB; redirects are rejected
to avoid forwarding credentials. HTTPS is supported with normal certificate checks.

Metadata-capable providers use the shared service's optional `ResponseProvider`
contract. They currently fetch fresh responses rather than discard metadata in the
legacy bars-only cache. Legacy provider caching is unchanged. The service preserves
source/timezone/adjustment and split evidence for both chart request paths. This API
does not promise authoritative corporate-action coverage or final daily-bar completeness.
