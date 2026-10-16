# SBI cash close/group reconciliation (#339)

Only explicit SBI accounting buckets change. Completed cash groups now sum their constituent close results: reported pretax net P&L takes precedence, without fee/tax subtraction. A reported close acquisition basis takes precedence over the fallback average for that close and its remaining acquisition estimate. Missing reported results use the existing SBI average-cost fallback, labelled as estimates in trade warnings. Ordinary open partial-close groups retain null `trade.net_pnl`; conversion groups retain existing zero-realization rules.

Cash accounting no longer rounds acquisition cost merely because a new date starts. Rounding remains at the applicable calculated cash disposal boundary. Actual timestamps keep their sequence; date-only records retain the existing date-level reconciliation and uncertainty warning, without inventing a chronology. The review group entry average is descriptive; completed P&L uses close results rather than a second weighted group formula.

For SBI margin-long history used specifically to derive genbiki transfer costs, an explicit broker-reported earlier close basis removes that acquisition cost from supported margin inventory. Unsupported residual costs fail regroup. Without evidence the existing estimate remains. Historical mark-to-market position replay is intentionally unchanged: the correction is limited to the conversion-basis calculation mode. Original execution prices/quantities remain intact. Reported cash results are preserved across unverified corporate-action boundaries; unverified returns remain unavailable.

Synthetic examples:

- Reported closes +120 and -30 produce completed net +90, even with recorded fees.
- 100 shares at 100.1 on one day plus 100 at 101.2 the next, then 200 sold at 110: fallback net 1,800, rather than an extra day-rounding result of 1,600 or a separate weighted review result of 1,870.
- Margin 100 at 6,000 plus 100 at 5,000; an evidenced close uses basis 6,000. Conversion of the remaining 100 transfers 500,000 before applicable costs, instead of the unsupported 550,000 average estimate. Conversion itself realizes zero.

## Reviewed historical repair

Back up first. `POST /api/v1/trades/regroup` with `account_id` and `preview:true` stages the actual regroup inside a transaction and rolls back. Review `changed_groups` (before/after money, basis, quantities and execution IDs), `preserved_trade_ids` and `removed_trade_ids`. Audit annotations, tags, setups, attachments, coach records and cash links against those IDs.

Apply with the same account and returned `preview_token`. The token covers calculated groups and executions after derived metadata. Changed evidence or market results reject the apply. Any removal of an existing trade ID is refused, so associations cannot be cascade-deleted; such a repair needs a separately reviewed association migration. All writes roll back on failure. Repeating an unchanged application is idempotent. Non-SBI calls without review parameters retain the prior contract. No historical data is automatically regrouped on startup or upgrade. Restore the backup if the reviewed result is unacceptable.

## Validation

- Full Go suite passed, including non-SBI grouping/fees/options/import/dedup and partial-close regressions.
- New tests cover reported precedence, per-close/group totals, cash rounding, actual-time close/re-entry, date-only fallback, fees, split protection, conversion basis, preview no-op, stale/missing token refusal, repeat apply, late failure rollback and preservation of execution IDs/fields plus review notes/journals. Removal of a legacy review ID is rejected atomically.
- Web static checks: zero errors (existing warnings). Client regroup types now expose preview/apply responses.
- A SQLite backup of the generated #338 QA dataset was copied to an isolated database. Preview found two changed groups and zero removed IDs, with no persistent changes. Applying the reviewed result twice preserved executions, trade IDs and the synthetic journal note.
- Codex built-in browser against the disposable API: list, detail drawer and full trade page all read back the corrected synthetic total 29,780 (before: 29,980), the estimate warning and retained note. Rendered drawer inspected; re-entry/read-back verified after reload.

No real export, production database, audit artifacts or screenshots were uploaded. No release, deployment or live repair. Mobile not validated; outside current fork scope.
