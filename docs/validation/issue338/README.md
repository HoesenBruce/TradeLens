# SBI reported cash close enrichment (#338)

The existing `sbi_margin_pnl` wire format now accepts ordinary `売却` rows from SBI realized-P&L detail reports, alongside margin repayments and supported delivery settlements. The legacy format/source name is retained for client compatibility.

Only stock executions with explicit `sbi:cash`, `sbi:margin-long`, or `sbi:margin-short` lot provenance and the matching reduce/side/settlement semantics qualify. Account broker names or account cash/margin labels never enable enrichment. Match identity is JST date, symbol, side/semantic, quantity and price. Summary rows are not executions.

A unique candidate is eligible. Multiple economically identical candidates require the complete source multiplicity and identical basis/P&L results; assigning those identical results makes no chronology claim. Otherwise the preview reports ambiguity. Differing existing values or conflicting duplicate source rows are conflicts. Invalid, ambiguous or conflicting rows block the whole commit with HTTP 409. Unmatched rows remain visible and are skipped. Overlapping/repeated reports leave matching metadata unchanged.

New metadata records broker-reported basis and realized P&L, the SBI report provenance and `pretax` tax basis. Imported P&L is used verbatim; no fee or tax subtraction occurs in enrichment. Original execution fields/IDs/dedup hashes and review identities are preserved. Application and regroup run in the existing transaction, including rollback after a late regroup failure.

## Validation

All fixtures were generated for tests. No real exports, database copies, amounts, screenshots or account artifacts were used or published.

- Complete `go test ./...` passed; targeted importer/API tests also cover metadata writes followed by a synthetic market failure and transaction rollback.
- Web local `vp check`: zero errors (existing warnings); `vp test run`: 203 files / 1,258 tests passed.
- Codex built-in browser, isolated API on :8091 and Web on :5188: cash preview, cancel/back and fresh re-entry, successful one-close enrichment, read-back via repeated preview, zero-write repeated application, ambiguous same-date/price fills and conflict refusal were exercised. Rendered success and refusal states inspected.
- Existing margin enrichment, delivery-settlement, generic CSV, IBKR, options, other broker, dedup and default partial-close tests passed in the full Go suite.

Historical executions can be explicitly supplemented after preview. Missing source evidence is left unavailable/unmatched. This issue adds evidence only; cash trade-total reconciliation is #339, and performance aggregation is #340. Legacy supplemental files are not made replayable by this change. Back up before historical recalculation. No live account repair, release or deployment was performed. Mobile not validated; outside current fork scope.
