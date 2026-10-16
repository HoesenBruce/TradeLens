# SBI partial-close realized money (#340)

Only explicit SBI execution lot/position semantics opt into the close-event
population. Cash, margin-long and margin-short reuse the accounting strategy and
conversion-basis preparation used by regroup; monetary reads do not persist any
derived execution metadata or trade changes. The grouping preparation was shared
rather than reimplemented. Event-to-review association uses execution ID lookups.

## Money consumers

| Consumer | Definition |
| --- | --- |
| Header amount / existing return numerator | `summary.realized_pnl` |
| Home performance net / annual goal | Same monetary field |
| Account detail amount / return | Account-only full-history monetary field; existing ledger denominator |
| Settings account cards | Same realized equity curve less its separate cash ledger |
| Realized equity and drawdown | SBI close events + existing non-SBI closed groups + unchanged separate cash flows |
| Home gross / fees | Explicitly labelled completed-group amounts |
| Reports, calendar/Day Review, setups and outcome ratios | Existing completed-group review population |
| Trade list/detail P&L | Existing lifecycle contract; ordinary open partial groups keep null net |
| Historical mark-to-market account value | Existing reconstruction semantics, unchanged |

Summary retains every existing review metric and adds separate realized money
and contribution count. Date-only SBI closes use the broker's Tokyo market date
for date filters; chart points use the beginning of that calendar date in the
requested display timezone solely for bucketing. Precise records keep actual
timestamps. Non-SBI date and aggregation semantics remain unchanged. Amounts
are normalized with the existing currency policy; funding denominators are unchanged.

No deposits, withdrawals, dividend/tax cash records or unrealized P&L enter trading
money. The curve still includes its existing separately defined cash-ledger flows.
Conversion-zero legs are omitted. A completed SBI group is replaced by its close
events, never added on top of them. Unknown source evidence is not invented.

## Synthetic before / after

- An open SBI position partially closes for -200. Review net/count stay 0/0;
  money is -200. A later close of +50 makes cumulative money -150 once.
- Add a generic completed +100 group: mixed money is -100 during the SBI partial,
  then -50 after final close. The generic contribution is always +100.
- With 1,000 funding in the SBI account: partial amount/return/equity are
  -200 / -20% / 800; after final close they are -150 / -15% / 850.
- A window containing only the final close has +50 monetary P&L and still one
  losing completed review group. A window excluding both closes gives zero.
- In an API test with 1,000 deposit, +10 dividend and -3 cash fee, partial equity
  is 807 while trading realized P&L remains -200. Ledger entries are not retaxed
  or added into trading money.

The API tests cover cash and margin long/short, conversion-zero, partial/final and
multiple closes, repeat full-history import, inclusive/exclusive dates including
New York display boundaries, empty/symbol/account/mixed-account filters, unchanged
review counts and readonly execution/trade records. Shared strategy tests cover
split and fallback protections plus existing IBKR/generic/option/broker semantics.

Codex built-in browser used an isolated API and generated database. Exercised:
partial close, mixed/SBI/generic/empty scopes, account filter read-back/re-entry
and reset, same-date inclusion, recent-date exclusion, date read-back, unfinished
custom-range cancellation and clean re-entry, final close, final-only +50 and
all-history -150, plus account cards/detail return/amounts. Rendered Home and
account detail states inspected. Source reports and database snapshots remained
local; no real trades or audit artifacts were uploaded.

Historical displayed money can change from raw SBI history on upgrade, without
rewriting stored trade/execution rows. Recalculating stored historical groups
still requires the #339 reviewed repair process. Monetary reads scan selected
execution history and reuse existing corporate-action checks; very large histories
or unavailable configured providers can make these reads slow or unavailable.
No new persistent aggregation cache was added.

Mobile not validated; outside current fork scope. No production repair, release
or deployment was performed.

Final checks: complete Go suite passed; the additional SBI FX and margin cases
passed separately. Web check: zero errors (existing warnings); full Web unit
suite: 203 files / 1,260 tests passed. A failing local synthetic market provider
also exercised the account-detail unavailable state (equity and money show —,
return hidden), followed by recovery to the normal offline QA configuration.
