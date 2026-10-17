# Issue #350 — SBI margin close P&L disambiguation

## Implementation already present on main

Base commit `27077d96` (PR #349) already calls `sbiConsistent` before deciding
whether an SBI credit close is ambiguous. It excludes candidates whose stored
`broker_reported_realized_pnl` disagrees with the report. Missing evidence remains
compatible, so it cannot be discarded to manufacture a unique match.

This follow-up adds regression coverage for a complete multi-row CSV import;
it does not duplicate the resolver or change accounting rules. Existing unit tests
cover missing P&L, conflicting evidence, margin-long/short, cash and non-SBI
isolation. Ambiguous opening allocations remain blocked by the existing resolver.

## Synthetic fixtures and API regression

Both `api/internal/importer/testdata/sbi-margin-pnl-disambiguation-*.csv` files
are entirely fictional. They contain eight date-only fills and four report rows:

| Symbol | Position | Closing quantity/price | Reported P&L | Acquisition basis |
| --- | --- | --- | --- | --- |
| 918A | Margin long | 40 at 1,250 | 1,960 / -840 | 1,200 / 1,270 |
| 919A | Margin short | 40 at 1,250 | 1,960 / -840 | 1,300 / 1,230 |

The report deliberately uses a different row order from the history.
`TestSBIMarginPnLDisambiguationPreviewCommitAndReplay` checks both report orders:

- Four distinct exact close matches and resolved opening allocations.
- Repeated previews write nothing.
- A report containing three valid rows and one conflicting P&L returns 409 and
  leaves all executions unchanged.
- Commit enriches exactly four closes, preserving execution IDs and trade totals.
- Repeated imports in either order preserve each match and acquisition basis,
  return `already_enriched`, and write nothing.

The regression test also failed as expected when a temporary Go build overlay
removed `candidates = compatible`: `enrichable` became `ambiguous_match`.
The overlay was never applied to repository sources.

## Local validation — 2026-10-17

Passed on the issue branch:

```sh
cd api
go test ./...
go vet ./...
go build ./...
```

Codex built-in browser acceptance used a disposable SQLite database, API on
`127.0.0.1:8091` and Web on `127.0.0.1:5174`. Market data, economic calendar
and background jobs were disabled. No production account or database was used.

Exercised history upload (8 inserted), report preview (`Ready to enrich: 4`),
Back/cancel (8 fills, no enrichment), file-selection reset, re-entry, report
commit (4 enriched, no errors), read-back (`Already enriched: 4`), repeated
commit (0 enriched, 4 skipped), and conflicting-P&L preview/commit refusal.
Both repeat and conflict attempts left persisted executions unchanged. Direct
read-only SQLite assertions confirmed the four correct bases and preserved
execution IDs, quantity, price, fees, commission, timestamps, dedup hashes and
original reported P&L.

No private source files, real transaction records, filenames or account data are
included in the issue, PR, fixtures or validation artifacts shared with GitHub.
No automatic migration or historical recalculation is introduced. Applying a
new enrichment retains the existing regroup/statistics behavior.

Mobile not validated; outside current fork scope. Deployment and validation
against a production account are outside this change.
