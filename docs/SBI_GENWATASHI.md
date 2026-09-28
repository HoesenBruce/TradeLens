# SBI 現渡 settlement (#123)

A history row becomes two linked executions: a cash sale and a margin-short
reduction. Both carry `event_type=position_settlement`,
`settlement_type=genwatashi` and the same `settlement_id`. Original price, fees,
receipt and dedup identities survive regrouping. Flat string audit fields record
cash quantity disposed, cash basis used, short quantity closed, disposal price,
applicable costs, realized P&L and its source on both legs.

SBI's [settlement formula](https://faq.sbisec.co.jp/answer/5edde2cb878c430011c178ce/)
is short entry price × delivered quantity minus aggregate expenses. Those expenses
include opening fees. `受渡金額/決済損益` on a 現渡 history row is a net receipt,
not short-cover P&L. If present, it determines net proceeds and total costs; it is
never charged fees again. Previously booked short-opening fees are reversed
proportionally when the aggregate settlement receipt is booked.

The cash disposal owns realized P&L. Cash 100 @900 and short 100 @1000 realizes
10,000 JPY; 600 aggregate costs makes it 9,400 JPY. The short settlement realizes
zero. Cash is costed with the existing SBI daily average and upward yen rounding,
not FIFO or an invented delivered-lot identity. Remaining short consideration is
reduced by the reported delivered entry price, without inventing individual lots.

An explicit `実現損益` and `平均取得価額` on the history row, or an unambiguous
matching SBI realized report's 現渡 row, takes priority. The realized-report
[help](https://search.sbisec.co.jp/v2/popwin/help/assets/profits_loss_domestic.html)
distinguishes 現渡 from ordinary credit repayment. A reported net result is not
charged settlement costs again. `realized_pnl_source` distinguishes
`broker_reported` from `calculated_sbi_cash_basis`.

Replay settles both positions atomically. Missing legs, insufficient holdings,
inconsistent pair metadata or uncostable short history produce
`invalid_genwatashi_settlement` warnings. Regroup/import rejects and rolls back
such cases rather than assigning a generic lot or fabricating an authoritative
result. Without a broker receipt, a delivered price inconsistent with the
reconstructed short basis is rejected. This deliberately limits support for
ambiguous partial deliveries from different short entries until source evidence
is supplied. Opening expenses exceeding the supplied aggregate settlement costs
are also incomplete evidence.

Only explicit SBI buckets opt in. Default/other-broker accounting and 現引 are
unchanged. There is no migration that reinterprets ordinary historical buys/sells.
Regrouping accounts containing these new events recalculates their derived
trade/calendar statistics and audit fields; imported execution values remain
unchanged. Position replay includes these events in historical account valuation.

Web execution tables, compact rows and detail drawers label both legs as 現渡.
Automated checks cover full/partial delivery, excess cash/short quantity, cash
averaging, authoritative P&L/receipt, opening expenses, broken pairs, remaining
short basis, account/position isolation, duplicate imports and rollback. Existing
Go regressions cover 現引 and non-SBI/default behavior. Built-in browser acceptance
uses a disposable real API: upload → preview → commit → closed cash/short trades,
9,400 net / 10,000 gross / 600 fees, ordinary opening labels, 現渡 labels in the
drawer and full detail, close/reopen and 390px compact rendering. No new editable
controls were added. Mobile was not validated; outside current fork scope.
