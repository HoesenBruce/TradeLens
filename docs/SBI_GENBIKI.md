# SBI genbiki history repair

On server startup, legacy SBI synthetic genbiki pairs are annotated and their
account's trades regrouped in one transaction. No execution is deleted, and IDs,
original prices, import batch links, dedup hashes, and journal links are preserved.
Already repaired accounts are skipped. Existing #130 conversions without derived
cash basis are regrouped as well.

Recognition requires matching user/account, batch (including the CLI's absent
batch), symbol, stock instrument, quantity, reference price and multiplier, explicit
SBI margin-long/reduce and cash/increase details, no existing conversion or
settlement P&L, and zero cash-leg costs. The decisive legacy-parser signature is a
margin leg at an even microsecond after Tokyo midnight followed exactly one
microsecond later by its cash leg. Ordinary source rows only have even offsets.
Rows whose precision or provenance cannot establish this signature are left alone.

The cash leg retains the source price and records `transferred_cost_basis` and
`transferred_unit_cost` in execution details. Grouping uses these freshly replayed
costs, never the conversion reference price. For example, 100 margin shares at
1,000 plus 500 conversion costs transfers 100,500; combining this with 100 cash
shares at 900 produces 200 shares with initial average cost 952.5. The conversion
day position snapshot preserves that average; subsequent SBI daily rounding and
sale costing still round the unit cost upward to a whole yen.

Margin opening fees are allocated proportionally to the converted quantity and
conversion costs are capitalized once. Conversion costs remain visible on the
original execution, but are excluded from the margin trade's realized fees. Pure
conversions have zero gross/net P&L; ordinary partial-close profits are preserved.
Cash trades containing conversions use SBI daily average-cost realized results.
These changes deliberately recalculate historical SBI conversion statistics;
non-SBI/default accounting is unchanged.

Missing margin history is not reconstructed. Regroup rejects an uncostable cash
conversion rather than substituting its reference price. Startup logs the failure;
the account repair transaction rolls back and its history needs review. Supply the
missing source history before retrying. This change was verified on disposable data, not user trading history.

## Verification

Automated tests cover legacy detection and negative matches, repeat repair,
unchanged IDs/dedup and re-import, multiple opening fills, partial/full transfers,
existing cash, opening/conversion fees, later and same-day cash sales, preservation
of ordinary partial-close P&L, API daily results, and existing broker regressions.

Built-in browser acceptance uses a disposable SQLite API and the working-tree Web
app: a legacy pair repaired on restart shows zero net/gross P&L, an auditable 500
execution fee, and combined cash acquisition value 190,500. The calendar and day
summary show zero, with no win or loss; opening, closing and reopening the detail
preserve the result. JPY price formatting displays 952.5 as 953; the API retains
952.5. No new UI controls or mobile changes were made. Mobile was not validated
and remains outside the current fork scope.
