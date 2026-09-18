# Historical Account Reconstruction — Design

**Issues:** #28 (specification), #12 (parent feature)

**Status:** Specification

## Goal and boundaries

Reconstruct one estimated end-of-day account state per Tokyo exchange session from
persisted executions, cash-ledger rows, and historical daily closes. This is an
estimate derived from TraderMemos data, not a reproduction of SBI's internal
accounting.

This specification defines the contracts for #29–#32. It does not implement position
replay, valuation, an API endpoint, persistence, or a Web chart. The existing
`GET /analytics/equity-curve` remains unchanged: it continues to report cash flows plus
closed-trade net P&L and does not become a mark-to-market account-value curve.

V1 uses the existing execution and cash-ledger records as source data, the existing
average-cost convention, and the shared `marketdata.Service`. It does not add a daily
snapshot table or a second price loader.

## Input and ordering rules

- Reconstruct each account independently in its base currency. Multiple selected
  accounts may be summed only when their base currencies match; mixed currencies are
  rejected rather than converted with a current FX rate.
- Replay all executions and cash events through the requested end date. The `from`
  filter limits output only: earlier events must still establish opening cash,
  positions, average cost, and cumulative P&L.
- Order executions by `executed_at`, then stable execution ID. Order cash rows by
  `occurred_at`, then stable cash-row ID. Apply all events for a Tokyo date before
  producing that date's end-of-day point.
- Partition executions by account, symbol, instrument, and explicit lot metadata. SBI
  cash, margin-long, and margin-short lots remain separate. Within a partition, use
  weighted-average cost, matching the current trade-grouping convention.
- Quantity and money calculations use the execution multiplier, defaulting to `1` only
  where the existing model does so. Round public money fields to two decimals after
  aggregating a point, not after each intermediate operation.
- Invalid sequences (for example, a margin-long close larger than the open
  margin-long quantity) do not invent an opposite position. The affected point is
  incomplete and includes an `invalid_execution_sequence` warning.

## Position replay

Let `q` be the absolute quantity, `p` the execution price, `m` the multiplier, `c`
the total execution fee plus commission, `a` the current weighted-average entry price,
and `P` the end-of-day close.

### Cash long

- A buy increases cash-long quantity and recomputes weighted-average cost. Its cash
  settlement is `-(q × p × m) - c`.
- A sell closes up to the available quantity at average cost. Its cash settlement is
  `+(q × p × m) - c`, and gross realized P&L is
  `(p - a) × q × m`.
- A partial sell preserves the remaining position and its average cost. A full sell
  leaves zero quantity. A sell beyond the available cash-long quantity is invalid in
  V1; V1 does not infer an unlabelled short sale.
- End-of-day value is `remaining_quantity × P × m`; unrealized P&L is
  `(P - a) × remaining_quantity × m`.

### Margin long

- An opening buy increases the margin-long quantity and average cost. No principal is
  removed from `cash_balance`; `c` is removed when incurred.
- A repayment sell realizes `(p - a) × q × m`. Cash receives that gross P&L and pays
  `c`; the margin notional itself is not booked as cash.
- Partial and full closes follow the cash-long quantity rules.
- Its contribution to `open_position_value` and `unrealized_pnl` is
  `(P - a) × remaining_quantity × m`, not the gross market notional.

### Margin short

- An opening sell increases the margin-short quantity and average entry price. Short
  proceeds are not treated as spendable cash; only `c` changes cash when incurred.
- A repayment buy realizes `(a - p) × q × m`. Cash receives that gross P&L and pays
  `c`.
- Partial and full closes follow the same average-cost rule.
- Its contribution to `open_position_value` and `unrealized_pnl` is
  `(a - P) × remaining_quantity × m`.

### `現引`

Use the canonical imported representation already produced by the SBI importer:

1. close the margin-long quantity at the conversion price, realizing margin P&L; then
2. open the same quantity as cash long at the same price.

Apply both legs in their persisted order on the same Tokyo date. Charge the conversion
fee once, on the margin-close leg. The cash buy pays the converted principal, while the
new cash holding contributes the same market value, so the transition does not double
count quantity, principal, or P&L. Missing or mismatched legs produce an
`invalid_genbiki_transition` warning and an incomplete point.

## Cash and account events

`cash_balance` begins at zero and includes every persisted cash-ledger amount exactly
once plus the execution settlements above. Account `starting_balance` is metadata and
must not be added separately because account creation already records it as an opening
`deposit` row.

Persisted ledger amounts are signed source-of-truth values:

| Type | `cash_balance` | `contributed_capital` | `realized_pnl` |
| --- | ---: | ---: | ---: |
| `deposit` | add amount | add amount | no change |
| `withdrawal` | add amount (normally negative) | add amount | no change |
| `dividend` | add amount | no change | no change |
| `interest` | add amount | no change | no change |
| `fee` | add amount (normally negative) | no change | add amount |
| `adjustment` | add signed amount | no change | no change |

SBI withholding tax, tax refund, and other broker cash corrections remain signed
`adjustment` rows in V1. They affect account value but not trading P&L or contributed
capital. A later typed-tax model may separate them without changing historical raw rows.

Execution fees and commissions reduce cash when their execution occurs and reduce
`realized_pnl` once. A separate ledger fee is an additional charge; importers must not
also create a ledger row for the same execution fee. If duplicate representation is
detected but cannot be resolved deterministically, return
`ambiguous_duplicate_fee` and mark the point incomplete rather than guessing.

`realized_pnl` is cumulative net trading P&L: gross P&L from closed quantities minus
all execution fees/commissions and ledger `fee` amounts incurred through the date. It
does not include deposits, withdrawals, dividends, interest, taxes, or adjustments.
Fees are recognized when incurred, including fees on positions that remain open.

## Daily values and formulas

For a complete end-of-day point:

```text
cash_balance = cumulative signed ledger amounts
             + cumulative execution settlements

cash_long_value = sum(cash-long quantity × close × multiplier)

margin_open_pnl = sum(margin-long unrealized P&L)
                + sum(margin-short unrealized P&L)

open_position_value = cash_long_value + margin_open_pnl

unrealized_pnl = sum unrealized P&L for cash-long, margin-long, and margin-short lots

estimated_account_value = cash_balance + open_position_value

contributed_capital = cumulative deposits + cumulative withdrawals
```

Withdrawals are normally negative, so the last formula is equivalent to cumulative
deposits minus withdrawal magnitudes. Trading P&L, dividends, interest, fees, taxes,
adjustments, and mark-to-market changes never change `contributed_capital`.

`open_position_value` deliberately means the positions' contribution to liquidation
value: gross market value for cash holdings, but only mark-to-market P&L for margin
positions whose principal/proceeds were never booked into cash.

## Market dates and prices

- V1 output dates use `Asia/Tokyo` calendar dates and the Tokyo exchange-session
  calendar. `from` and `to` are inclusive market dates.
- Produce at most one point per exchange session. Do not create weekend or exchange-
  holiday points. Events on a non-session date are included in the next session's
  point; they remain ordered by their actual timestamp.
- Value a position held at session end with that instrument's daily **unadjusted**
  close whose `market_date` equals the point date. Reject adjusted or unknown-
  adjustment series; never mix adjusted prices with unadjusted broker quantities.
- Fetch bars only through the shared market-data abstraction. Preserve the provider,
  source, timezone, and adjustment status in response metadata/warnings.
- A position opened and closed before the session close contributes realized P&L and
  cash settlement but no end-of-day open value.

### Missing-data policy

No arbitrary substitution, interpolation, zero price, current quote, purchase price,
or adjusted close is allowed.

| Case | Behavior |
| --- | --- |
| Weekend or exchange holiday | No point and no price lookup. |
| Confirmed suspended instrument with a prior valid close | Carry that close forward and emit `carried_forward_suspension_price`. |
| Expected session bar missing for an unexplained reason | Point is `incomplete_missing_price`; affected monetary totals are `null`. |
| Newly listed instrument before its first valid close | If no position exists, ignore it; if a position exists, use `incomplete_missing_price`. |
| Delisted instrument after its last known close | Do not carry indefinitely; use `incomplete_missing_price` until an explicit terminal-value rule exists. |
| Provider fails before any usable result | Return the normal API error envelope with HTTP `502`; do not return fabricated points. |
| Provider fails for only part of a multi-instrument request | Return points, but affected points are `incomplete_missing_price` with a provider warning and nullable totals. |

Carry-forward is therefore allowed only when suspension is positively identified, not
merely because a bar is absent.

## Corporate-action boundary

Raw executions and ledger rows are immutable evidence. V1 never rewrites their prices,
quantities, or timestamps and never applies an inferred split ratio.

When #14 reports an unconfirmed or unsupported corporate-action candidate between the
last trustworthy state and a requested point, that point and subsequent affected
points use `unsupported_corporate_action`. Their valuation-dependent monetary fields
are `null`, and warnings identify the instrument, effective date, suspected ratio when
available, and candidate reference. Reconstruction may resume only after an explicit,
auditable adjustment/confirmation rule is implemented; crossing another price does
not silently clear the state.

Before #14 is available, a detected adjusted/unadjusted mismatch or otherwise known
corporate-action boundary follows the same unsupported path. Absence of a detector is
not permission to infer or apply a split.

## Additive API contract

Add a separate authenticated endpoint; do not change `/analytics/equity-curve`:

```http
GET /api/v1/analytics/account-value?account_id=<id>&from=YYYY-MM-DD&to=YYYY-MM-DD
```

`account_id` follows existing analytics filtering. Omitting it selects all eligible
non-backtest accounts with the same base currency. `from` and `to` are inclusive Tokyo
market dates. The response is additive and uses nullable monetary fields whenever a
point is not safely valuatable:

```json
{
  "currency": "JPY",
  "timezone": "Asia/Tokyo",
  "adjustment_status": "unadjusted",
  "points": [
    {
      "date": "2026-09-18",
      "estimated_account_value": 1234567.89,
      "contributed_capital": 1000000.00,
      "cash_balance": 345678.90,
      "open_position_value": 888888.99,
      "realized_pnl": 12345.67,
      "unrealized_pnl": 45678.90,
      "status": "complete",
      "warnings": []
    }
  ]
}
```

Allowed point statuses are:

- `complete`: every held instrument has a permitted price and replay is valid;
- `incomplete_missing_price`: replay is valid but at least one required price is not;
- `incomplete_invalid_sequence`: execution or fee data cannot be replayed safely;
- `unsupported_corporate_action`: the point crosses an unsupported boundary.

Warnings are structured and stable enough for Web rendering:

```json
{
  "code": "missing_price",
  "instrument": "6501",
  "date": "2026-09-18",
  "message": "No unadjusted close is available for this Tokyo session."
}
```

For any non-`complete` point, `contributed_capital`, `cash_balance`, and
`realized_pnl` may remain populated when they are deterministic. Set
`estimated_account_value`, `open_position_value`, and `unrealized_pnl` to `null` if
any required position valuation is unsafe. Do not return partial totals that look
complete.

Normal authentication, invalid-filter, account-isolation, and mixed-currency failures
use the repository's existing error envelope and status-code conventions. An empty
valid range returns `points: []`.

## Required implementation checks

Later issues must leave deterministic checks for:

- deposit → cash buy → price change → partial sell → full sell;
- margin-long and margin-short open, partial close, and full close;
- `現引` value continuity and single fee charging;
- an open position established before `from`;
- withdrawal, dividend, interest, fee, tax/refund adjustment, and signed adjustment;
- weekends and exchange holidays producing no fabricated point;
- confirmed suspension carry-forward versus unexplained missing close;
- provider failure and suspected corporate-action states;
- account isolation, mixed-currency rejection, and inclusive date bounds;
- backward compatibility of `/analytics/equity-curve`.

Web implementation in #32 must render the two labelled series, surface warnings and
incomplete/unsupported states, and satisfy the repository's Web end-to-end rules.
Mobile UI remains outside the current fork scope; shared contracts must remain
platform-neutral, but mobile is not considered validated.
