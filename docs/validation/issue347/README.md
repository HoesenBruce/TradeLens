# Issue #347: SBI settlement evidence and conditional FIFO

Initial audit baseline: `f29388f9d60557d557379be59372ef88d7041b6b`
(`origin/main`, fetched 2026-10-17). Final integration baseline:
`60f462a04c062ac7604bcf99ba083330d303ac98` (#348 merged during acceptance).
The branch incorporates that main commit with a regular merge, without rewriting history.
Branch: `fix/347-sbi-settlement-fifo`. Synthetic data only; temporary SQLite only.

## Baseline audit and scope

The primary checkout had uncommitted #346 UI work. An existing #347 worktree had
an uncommitted 15-line settlement filter and an untracked test, with no PR.
Both were inspected and preserved; this isolated worktree starts from the fetched
main baseline. No reset, clean, force push, historical database rewrite or deployment.

Reviewed #347's latest body (no comments), closed #307/#308 and merged #309/#310,
recent #338/#341, #339/#342, #340/#343 and #344/#345. Open Draft #313 retains import
sources and overlaps parser/commit files; this change does not depend on it, add
its schema, or implement its reprocessing workflow. Resolve those small source
metadata conflicts explicitly when integrating the branches.

Existing executions store account/user, symbol, direction, quantity, price,
fees/commission, timestamp, date precision, explicit SBI position bucket and
reported settlement P&L/basis in `details`. Review associations use the existing
trade-execution link table. Historically, original pre-offset times, credit
class, opening-date evidence and itemized charge terms were not consistently
retained. Do not manufacture them for existing rows.

## Matching contract

There are two different links, exposed separately:

- `match_provenance`: report row to repayment execution. `exact` means one
  consistent SBI close after filtering on supplied settlement results, basis and
  retained terms. Unknown evidence does not count as a contradiction. Multiple
  remaining repayment records stay `unresolved`, even when a random-ID assignment
  could make their counts fit. The cash matcher retains its existing behavior.
- `opening_match_provenance`: repayment to supported opening inventory. `exact`
  requires a unique consistent opening. An average basis at an interior price
  cannot prove a unique lot: different-price mixtures could have that average.
  Broker opening date/price/class can further narrow candidates. Original evidence
  takes precedence over opening time order.
- `fifo_inferred` applies only when all competing live openings are equivalent:
  same user/account, stock, explicit margin direction/bucket, market-local opening
  day, original price, credit class and available terms, equal canonical and raw
  per-unit charges, and enough remaining supported quantity. Credit class and
  fees/interest/lending/other-charge values must be explicit, including zeroes.
  Prices/basis/quantities have no UI tolerance in credit matching; retained decimal
  evidence and raw charge ratios use `math/big.Rat`.
- Other cases retain `unresolved` opening status and block the entire credit
  enrichment batch, with import-history/evidence diagnostics. Unsupported inventory
  cannot create an opposite-direction position. A cycle becomes usable again only
  after its quantity ledger demonstrably returns to flat.

FIFO uses the original opening timestamp captured before existing storage offsets.
Dates remain dates; no original clock time is stored for a date-only source.
Identical timestamps or date-only openings form equivalent groups. An allocation's
`execution_ids` denotes the whole indistinguishable group and its aggregate
quantity; member IDs are sorted only to serialize the group. There is no arbitrary
per-member residual or assertion of SBI's actual repayment designation. Equal-time
repayment records without an evidenced independent allocation remain unresolved;
their combined demand cannot overdraw an opening depending on input order.

Both partial closes and one repayment spanning equivalent groups conserve quantity.
Residual inventory descended from a FIFO allocation remains inferred until flat;
it cannot become broker-exact just because one candidate remains.
Earlier repayments consume availability before later ones; unresolved earlier
history prevents speculative subsequent allocations within that open cycle.

## Persistence, accounting and historical safety

No schema or migration. New SBI imports retain a small allowlist of original
settlement/credit fields in the existing `details.sbi_source` JSON string. Original
price/quantity decimal text and actual pre-offset times are retained. Original
date-only inputs retain date precision. Existing dedup keys and timestamps are
unchanged. There is no startup rewrite or unknown-to-exact migration.

New successful credit enrichment persists `sbi_report_match_provenance`,
`sbi_opening_match_provenance`, and `sbi_opening_allocations` (JSON string) alongside
the existing reported basis/P&L fields. Existing source amounts are never replaced.
Commit revalidates the entire account snapshot, including allocations, before
writing. HTTP import uses the existing transaction, so any unresolved row or failed
regroup rolls back its batch and changes.

Allocations are provenance, not a replacement accounting engine. Existing SBI
settlement P&L remains authoritative for realized-close accounting, without another
fee/interest deduction. Position replay, monetary exposure, acquisition cost,
review IDs/links and default/non-SBI partial-close semantics remain on their existing
paths. In particular, historical account-value replay retains its existing
allocated-fee valuation model; this issue does not normalize its calculation to
settlement accounting. The fee-bearing regression asserts both the unchanged
valuation snapshot and exact broker-reported realized-close result separately.

Review warnings reuse the existing Web warning surface. Inferred allocations are
explicitly labelled TradeLens inference, and precise holding duration is unavailable
for them. Date-only review rules remain in force. No UI component redesign.

For existing accounts, back up first and inspect an enrichment preview. Missing
original credit/charge evidence may require source reconciliation; repeating an
old import does not silently overwrite its retained metadata. Do not delete history
or fabricate missing fields to force FIFO. Enrichment rollback remains a full
verified database-backup restore, consistent with the existing enrichment policy.

## Validation

- Complete `go vet ./...`, `go test ./...`, `go build ./...`.
- Relevant race suite: importer, trades and API with SBI, default, IBKR, partial,
  regroup and cross-user regression cases.
- Web check (zero errors; inherited lint warnings), 206 files / 1314 unit tests, typecheck,
  production build. No dependencies added.
- `sbi_matching_test.go`: unique settlement evidence in long/short buckets, unknown
  and contradictory results, different prices/dates/classes/fees/interest, missing
  terms, insufficient inventory, account/direction isolation, source decimal values
  beyond float display precision, partial/spanning closes, consumed residuals,
  equal-time/date-only groups, shuffled inputs, conflicting duplicated reports,
  original exact matches and non-SBI exclusion.
- Temporary-SQLite persistence test: nonzero charges, original source preservation,
  full position/cash/valuation snapshot equality, reported net P&L 1497 with no
  second fee deduction, remaining quantity 50 at basis 100, repeat commit zero
  writes, and rejection of tampered allocation metadata before any write.
- Real HTTP integration test: Preview is read-only and agrees with Commit;
  pending valid+unresolved batch writes nothing; execution IDs/times/quantities/
  prices/fees/dedup keys and trade identities survive; repeat preview/commit/import
  is idempotent; conflict and missing-close batches return 409 atomically.
- Playwright against an isolated running API: preview, cancel, commit, read-back,
  full detail/reload, repeat enrichment, conflict refusal; existing SBI direction
  regression uses current progress/date labels and an explicit Tokyo timezone.
  A resource-contention timeout in an unchanged drawer unit test was resolved by
  rerunning the complete unit suite with two workers, without changing assertions
  or its timeout. An existing replay test also compared `updated_at` across
  regroup operations; it now asserts monotonic audit time separately while still
  comparing every business field (including original opening/closing times).
  Built-in browser also
  visually verified the warning, unknown exact duration and 200 opened / 150
  closed / 50 remaining. Screenshots contain only generated data.

All 18 requested case categories are represented by these tests and the retained
SBI importer/semantics/dedup and default-broker accounting regression suites.

Limitations: incomplete historical credit/charge evidence remains unresolved;
ambiguous close identities and uncertain same-day repayment sequence are not
reconstructed. Source decimal evidence protects new matching while existing
canonical monetary/quantity fields retain their current numeric representation.
Mobile and PostgreSQL runtime acceptance were not run; no schema/SQL changed.
No production dataset, historical repair, release, merge or deployment was performed.
