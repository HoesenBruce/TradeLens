# Issue #312 — retained CSV source replay

Validated locally on 2026-10-15 from `feat/312-import-reprocessing`, based on main
`3d383d89`, in its own Codex worktree. All data and credentials were synthetic;
no production database, release or deployment was touched.

## Implementation boundaries

- Execution CSVs: generic, SBI, journal/round-trip, date-order/timezone choices,
  option overrides and multi-leg SBI conversions. Source row values include unused
  columns, and original row offsets are retained for SBI's date-only timestamps.
- Unique source content/context/execution identities is retained once per account.
  Revisions supersede earlier rows; unrelated rows remain replayable from their
  original batch. Existing execution dedup/accounting semantics remain unchanged.
- No original file bytes. Rejected/non-fill rows, JSON, HTML/XLSX, cash-only and SBI
  enrichment replay remain outside this implementation. UI explains unavailable
  sources. Legacy source content cannot be reconstructed without another upload.
- Existing trade IDs and execution membership must survive; otherwise the entire
  replay is rejected. Manually edited/enriched/deleted fills are never overwritten
  or resurrected. Existing imported journal annotations are not reapplied.
- Preview writes occur only in a transaction that is rolled back. Apply verifies
  a fingerprint covering current source/fill/trade data and projected results;
  serializable transactions detect concurrent writes. Unexpected DB/provider error
  details are not returned to the browser.
- Replay remains synchronous; very large remote batches may exceed request deadlines.
  Database backup/recovery includes the retained source tables. No expiry/cap or
  automatic historical rewrite is introduced.

## Automated validation

| Check | Result |
| --- | --- |
| `go test ./...` | PASS; final subsequent changes also passed relevant replay/import/store tests below |
| `go vet ./...` | PASS; final API/store/importer vet also passed |
| SQLite replay API regressions | PASS |
| PostgreSQL 16 replay API + database conformance | PASS against disposable local Docker database |
| 1000-row source archive chunking and duplicate upload | PASS; 1000 fills/source rows/links, one context after repeated upload |
| Archive write failure | PASS; fills, trades and all source tables roll back together |
| Old-parser fee omission simulation | PASS; raw rows repair fees and preserve execution IDs |
| Notes, risk, setups, tags, attachments and stored review links | PASS; unchanged association tables after replay |
| Manual edit, missing leg, grouping change, bad parser row, foreign user, reversed batch | PASS; all-table state unchanged after rejection |
| Preview cancellation, repeat apply and stale preview | PASS; no writes on preview; idempotent replay; calculation changes require new preview |
| Generic/SBI/journal/round-trip/genbiki/option choice/date-order replay | PASS |
| Provider error confidentiality | PASS; synthetic private error absent from response, all tables unchanged |
| Full Web suite | PASS: 199 files / 1244 tests |
| Final related Web/i18n tests | PASS: 4 files / 19 tests |
| `vp check` and `tsc -b --noEmit` | PASS; existing repository lint warnings remain |
| `vp build` | PASS |

The related replay command uses `go test ./internal/api -run TestReprocess -count=1`;
PostgreSQL is enabled by `TM_TEST_DATABASE_URL`. Database conformance uses
`go test ./internal/db -run Conformance -count=1`. No new dependencies were added.

## Codex built-in browser acceptance

Used the worktree Web server at `127.0.0.1:5178` and its disposable API at
`127.0.0.1:8091`, with a fresh QA database. The old custom server/session was explicitly
replaced before testing. Market data was disabled for deterministic same-day fixtures.

| Case | Evidence |
| --- | --- |
| Real CSV upload, mapping, timestamp preview and commit | Two fills inserted via Web; unused source columns retained |
| Reprocess preview | Two retained rows / two fills / two changed fills / one changed calculated trade |
| Cancel + re-entry | Fees remain zero and P&L 100; reopening requires a new preview |
| Apply + read-back | Fees become 2 + 3; P&L 100 → 95 and balance 10,100 → 10,095 |
| Repeat preview/apply | Zero changed fills/calculated trades; no duplicates |
| Review read-back | Same trade ID, saved Chinese note, risk 50 and execution grade A+ remain visible |
| Manual-edit error | Clear conflict, retry requires new preview; all database tables unchanged |
| Legacy / unsupported / reversed history | Unavailable source explanation shown; no misleading reprocess button |

The omitted-fee state was deliberately seeded to simulate an older parser, rather
than claiming a test of a historical release binary. Legacy/unsupported/reversed
history states were synthetic metadata fixtures; supported CSV upload itself went
through the real Web import flow. Reset/toggle cases do not exist for this action.

Screenshots: [preview](preview.jpg), [applied](applied.jpg),
[conflict](conflict.jpg), [review read-back](review-readback.jpg).

Mobile not validated; outside current fork scope. No merge, release or deployment.
