# Default reversal fees and historical regrouping

Issue #268 changes only default, non-SBI trade grouping. A fill that crosses zero
closes one position and opens the opposite position; its combined fees/commission
are split by the quantities closed and opened. The closing allocation is rounded
once to cents and the opening allocation receives the remainder, conserving cost.
Execution fees and quantities are not rewritten.

For example, buy 100 at 10 with cost 3, then sell 150 at 12 with cost 6:
previously the closed long had cost 9 / net P&L 191 and the open short had cost 0.
Now the closed long has cost 7 / net P&L 193 and the open short (50 shares) has
cost 2. The inverse short-to-long transition uses the same quantity rule.
Close-only fills, scaling, and existing default partial-close NetPnl semantics
remain unchanged. SBI cash/margin/現引/現渡 use their existing strategy and review
trade cost allocation; account cash/margin labels do not select this fix.

Manual execution creation rejects missing, null, zero, or negative quantity with
HTTP 400 before any write, matching the existing positive-quantity edit contract.

## Existing history and rollback

There is no startup migration, new bulk-recompute endpoint, or automatic production
history rewrite. Stored trades change only when their account is regrouped. Existing
imports, execution creation/edit/deletion, and sync can already trigger account-wide
regrouping, so their next write may update old reversal trades as well. Closed-trade
P&L/statistics can change by the cost moved to an open opposite position; after both
trades close, their aggregate cost and aggregate net P&L stay conserved. Stable trade
IDs and annotations are preserved when the underlying execution sequence is unchanged.

Before intentional historical regrouping:

1. Pause writes/sync and take a consistent full database backup (including SQLite
   WAL through the supported backup flow, or a database-native PostgreSQL backup).
   Keep the pre-change application version and a trade/statistics export for comparison.
2. Rehearse on a restored copy. Compare reversal costs, closed-trade P&L, open positions,
   annotations, and SBI/account-value results before permitting production writes.
3. If rollback is needed, stop writes, restore the full pre-change backup, and run
   the prior application version. A code-only downgrade leaves already regrouped
   rows unchanged until another regroup; restoring a backup also discards subsequent
   writes, which must be reconciled deliberately.
