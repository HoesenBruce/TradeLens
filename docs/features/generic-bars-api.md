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

Validation-capable archives should expose optional RFC3339 `fetched_at` on each
bar (actual source acquisition time), or on the response when one source snapshot
covers all bars. Reading an archive over HTTP must not advance these timestamps.
Missing provenance remains unknown and produces an incomplete validation; it does
not prevent chart display. This is an additive v1 field.
Optional `corporate_actions` uses the shared candidate shape (`effective_date`,
`candidate_type`, `suspected_ratio`, `status`, `source`, `evidence`). Explicitly
reviewed `rejected` candidates suppress only an exact date/type/ratio detection
match. Cash-dividend evidence is retained without converting price return to total
return. Unknown events never imply verified corporate-action completeness.

## Optional provider routing (#77)

Set `TM_MARKET_DATA_PROVIDERS=http,yahoo` (ordered, comma-separated names) to try
HTTP first, then Yahoo. Supported names are `http`, `yahoo`, and `finnhub` in any
order. This overrides `TM_MARKET_DATA_PROVIDER`; leave it empty to preserve the
existing single-provider selection and cache behavior. The Docker Compose service
forwards these variables. Explicit routing rejects unknown/duplicate/empty entries,
an invalid/missing HTTP base URL, and Finnhub without `TM_MARKET_DATA_API_KEY` at
startup; it never silently substitutes Yahoo for a configured Finnhub route.

The router lives behind `Provider` / `ResponseProvider`; consumers use the existing
service. It returns the first nonempty successful response, without merging bars.
Fallback occurs only for:

- A successful, explicitly empty response (HTTP `bars: []`, Finnhub `no_data`, or
  Yahoo's 404, explicit `Not Found` error, or empty result/timestamp array).
  Yahoo 422 is terminal because it can indicate an invalid request range.
- Explicit temporary unavailability: HTTP 429, 502, 503, or 504 from any adapter.
- `ErrUnsupportedResolution` (Generic Bars 422 with `unsupported_interval`).

All other failures stop routing: authentication/permission errors, invalid requests,
unknown server errors (including 500), malformed/partial data, transport/TLS/timeout
errors, and cancellation. Transport errors deliberately remain terminal because
these also include configuration/security failures. Unknown Yahoo/Finnhub payload
errors are errors rather than empty windows; malformed Finnhub arrays cannot panic.
No retries, source scoring, or load balancing are added.

If every provider returns empty, return the first empty response and its metadata.
If no provider supplies data and any fallback-eligible error occurred, return an
error instead of claiming a successful empty window. Cancellation stops the chain.
`provider` identifies the actual adapter, never `router`; HTTP `source`, timezone,
adjustment status, acquisition timestamps and corporate-action evidence are
preserved. Legacy providers report their own name as `source` and their fetch start
as acquisition time. An absent HTTP source or acquisition time stays unknown.

Routing uses fresh metadata-bearing responses and bypasses the legacy bars-only
cache, including daily transaction coverage and validation refresh. This avoids
returning cached bars from a different route or losing source metadata; enabling
routing can increase upstream requests. Single-provider caching is unchanged when
routing is unset.

**Deferred sub-step:** interval-specific route overrides. This first version uses
one deployment-wide order for every interval; it does not hard-code daily or
intraday preferences. A later change can add per-interval order configuration at
the router boundary without modifying chart, News Thesis, or analytics consumers.

Verification uses local HTTP fixtures and service tests for order, empty/error
exhaustion, terminal errors, cancellation, startup configuration, and provenance
through chart/transaction/validation paths. No Web UI is changed; live provider
availability and iOS/Android behavior are not validated by these tests.
