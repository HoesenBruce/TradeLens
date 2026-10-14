# Issue #308 — normalized trade quantity visibility

All acceptance data and screenshots are synthetic. Web is the delivery target;
iOS and Android were not validated.

## Read contract

Trade list/detail share `toTradeDTO`, which now adds `quantities` while preserving
`qty_opened`, `qty_remaining`, all monetary fields and execution contracts.

| Field | Meaning |
| --- | --- |
| `entry_qty` | Cumulative reconstructed opened quantity |
| `exit_qty` | Reconstructed entry quantity minus reconstructed open quantity |
| `open_qty` | Persisted unclosed quantity; nonnegative magnitude, direction supplies sign |
| `remaining_pct` | Open / entry × 100, for known positive entry |
| `lifecycle` | `open`, `partially_closed`, `closed`, or `unknown`; separate from P&L result |
| `unknown_reason` | Machine-readable reason for null quantity values |

The arithmetic runs once at the API read boundary on the broker-aware reconstructed
inventory. Web consumes this contract, rather than matching fills or summing buys
and sells. A reversal's closing and opening trades use their own inventory; a
700-share reversal execution is not counted twice as each trade's entry/exit.
Approved split/reverse-split normalized trade quantities use the same share units.
No raw pre/post split execution quantities are divided on the client.

Known open-only trades show entry N, exit 0, open N, 100%; known round trips show
entry N, exit N, open 0, 0%. Missing/excess SBI opening evidence, nonpositive entry,
inconsistent stored inventory (including an open trade with a zero remainder),
or an unverified corporate-action boundary returns null quantity values. These
render as **—**, with a localized explanation; zero is reserved for known zero.
The source fills remain visible, including supported disposal quantities.

Quantities describe a single reconstructed trade, not its account's aggregate
position, cash, market value, or signed monetary exposure. Existing accounting
precision/normalization is the source of truth; no importer, matching, cost basis,
fee, P&L, snapshot, dedup, account-value or split-adjustment policy was changed.

## Web behavior

The existing quantity column ID is retained for saved view/sort compatibility.
Its display is `Open Qty / Entry Qty` and it sorts by authoritative open quantity.
Unknown inventory sorts last in both directions in the paginated list; the optional
Position column uses the same open quantity. Compact Web displays the same pair
with an accessible explanation and a visible legend.

The drawer and full detail share the same four quantity tiles and position
lifecycle label. Numeric P&L outcome labels remain independent of position state;
an open trade with supported positive P&L keeps WIN rather than being relabelled
as a position lifecycle. Entry/exit prices and other financial metrics retain their
existing behavior. English, Chinese, Japanese, Korean and Hong Kong Chinese labels
and unknown-reason explanations are included.

New servers return this additive read model for existing rows immediately. There
is **no migration, forced regroup, reimport or production rewrite**. Older clients
can ignore it; a new Web client on an older server shows unavailable quantities
with a server-compatibility explanation. This feature does not repair legacy
misgrouped records without an existing warning; deliberately repairing accounting
history remains the separate #307 workflow.

## Validation

- API tests: 500→400, pyramiding 500→400→+200, full close 500, short 70→60,
  reversal 500→sell700, open-only, normalized split/reverse split, date-only and
  missing/inconsistent inventory. List/detail read-back agrees and legacy fields,
  fees and P&L are preserved.
- Existing SBI cash/margin, genbiki/genwatashi, account-value, split, IBKR/options,
  generic/manual and dedup regression suites remain in force.
- Web tests cover explicit quantities, unknown/old-server rendering, remaining
  percentages, pagination sorting with unknown-last semantics and independent
  outcome/lifecycle labels.
- Real disposable API browser test seeds Manual and IBKR default-path accounts,
  imports generic CSV and SBI date-only fixtures, checks same-symbol/account and
  SBI bucket isolation plus genbiki quantities, list/detail/reload/re-entry,
  sorting/cancel/reset, filters, English/Chinese/Japanese, long/short/closed/unknown,
  and 390px compact Web without horizontal page overflow.

## Reproduction

Start the API from this working tree on :8092 with a **new disposable SQLite DB**,
`TM_JWT_SECRET` set to a disposable secret, `TM_JOBS_ENABLED=false`,
`TM_MARKET_DATA_ENABLED=false`, `TM_ECON_CALENDAR_ENABLED=false`, and
`TM_CORS_ORIGINS=http://localhost:5174`. Start Web on :5174.

```sh
cd web
PLAYWRIGHT_BROWSERS_PATH=<installed-browser-path> \
E2E_BASE_URL=http://localhost:5174 E2E_WEB_URL=http://localhost:5174 \
E2E_API_URL=http://localhost:8092/api/v1 \
node_modules/.bin/playwright test e2e/trade-quantities.spec.ts
```

The test creates a synthetic setup user on an empty API and an isolated account
per broker per run. Existing API credentials may be supplied through `E2E_EMAIL`
and `E2E_PASSWORD`, but use a disposable instance only. Market data is disabled;
charts are outside this presentation feature's acceptance scope.

## Recorded checks and evidence

Go full tests and vet passed. Web check and TypeScript passed; the final Web unit
suite includes 196 files / 1,131 tests. Chromium acceptance passed against the
current compiled API and disposable SQLite database. The built-in browser was
unavailable; Playwright was used. No production data was accessed or regrouped.

Screenshots inspected: [list](list.png), [partial close](partial-detail.png),
[short](short-detail.png), [closed](full-detail.png), [unknown detail](unknown.png),
[unknown sorting](unknown-list.png), [compact list](compact-list.png) and
[compact detail](compact-detail.png). Compact Web is browser viewport validation,
not iOS/Android application validation.
