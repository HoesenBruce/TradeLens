# Issue #350 — SBI close-result enrichment without invented opening lots

## Behavior

SBI credit reports first match closing executions using date, symbol, direction,
quantity, price and compatible stored settlement evidence, including reported P&L.
Opening allocation is a separate provenance calculation. A matched reported result
can be enriched while opening provenance remains `unresolved`; it is not changed
back into an ambiguous close match merely because no opening lot can be proved.
Unknown openings have no allocations and retain a visible warning. Unsupported
holdings are not reconstructed, and existing unavailable P&L/return protections
remain in force. Exact lot-level duration is suppressed; date-only calendar
holding days retain their existing meaning.

Complete sets of indistinguishable credit results can be assigned as `equivalent`
close provenance. The compatible report count must equal the candidate count and
all results/evidence in that group must agree. Stored evidence can partition
otherwise identical closes into distinct P&L groups. A missing evidence value
cannot be discarded to manufacture uniqueness. Partial groups, different bases,
conflicting P&L, invalid rows and unmatched credit closes still block atomically.

## Synthetic fixtures

All CSV fixtures and public examples are entirely fictional.

- `sbi-margin-pnl-disambiguation-*.csv`: eight date-only fills and four long/short
  closes sharing quantity/price but having distinct P&L and acquisition bases.
- `sbi-margin-pnl-unresolved-*.csv`: thirteen fills and six report rows:

| Symbol | Scenario | Expected provenance |
| --- | --- | --- |
| 920A | Long openings at 900 and 1,100; reported average basis 1,000 | Exact close, unresolved opening |
| 921A | Short openings at 900 and 1,100; reported average basis 1,000 | Exact close, unresolved opening |
| 922A | Two identical 1,470 results plus a distinct 570 result at the same quantity/price | Two equivalent closes and one exact close |
| 923A | Supported opening quantity 20, reported closing quantity 40 | Exact close, unresolved opening; missing-history protections retained |

## Automated regression coverage

- Importer tests prove that non-equivalent/insufficient openings cannot invent
  allocations, while their matched reported results can be stored.
- Multiplicity tests cover long/short, candidate order changes, partial/oversized
  reports, different bases and overlapping unknown P&L evidence.
- API tests run both CSV pairs in both report orders: preview without writes,
  successful commit, persisted provenance, stable execution/trade identities and
  totals, and repeat imports in either order without writes. A mixed report with
  conflicting P&L is refused before any enrichment is written.
- Persistence tests verify correct reported bases, preserved original execution
  fields and P&L, unchanged valuation replay, null unknown allocations and repeat
  commit idempotence.
- Trade tests preserve reported money while hiding exact duration for unresolved
  openings, including timestamped executions. Default/IBKR accounting ignores the
  SBI provenance marker. The full suite retains existing cash, settlement,
  options, commissions and generic-importer regressions.

## Local acceptance — 2026-10-19

Passed:

```sh
cd api
go test ./...
go vet ./...
go build ./...
cd ../web
pnpm run check
pnpm test src/app/screens/ImportView.test.tsx
pnpm run build
```

The Web import unit test file passed all 20 tests. Static checks passed with
existing repository warnings. Go and Web builds passed.

Codex built-in browser acceptance used a disposable local SQLite database/API
and a synthetic account. Market data, economic calendar and scheduled jobs were
disabled. Exercised the unresolved/equivalent fixtures end to end:

1. History upload inserted thirteen fills, with no errors.
2. Preview showed six ready rows, including explicit unknown-opening and
   equivalent-close notices. Back/cancel wrote no enrichment and reset file
   selection. Re-entry retained the same effective state.
3. Commit enriched six closes. SQLite assertions verified correct bases,
   unresolved/null opening allocations, and preserved original execution fields
   and broker-reported P&L.
4. Read-back showed six already enriched rows with the same provenance notices;
   repeat commit enriched zero and changed no executions.
5. Conflicting P&L remained a visible conflict and was refused. A partial repeated
   close group remained ambiguous and was refused. Both preserved the stored data.
6. Trade detail and page reload retained the unknown-opening warning and source
   date precision, while preserving reported money and calendar holding days.

The original private files were also rechecked end to end only in a disposable
local database, including commit, unchanged execution identity/original P&L and
idempotent repeat. No private files, filenames, records, financial values,
account data or screenshots are included in published fixtures or evidence.

## Historical and platform impact

No schema migration, automatic backfill or historical rewrite is introduced.
Applying a report uses the existing regroup/statistics path, retaining execution
and review identities and existing valuation/P&L behavior. This change does not
recover missing opening history or assert SBI-designated physical lots.

Mobile not validated; outside current fork scope. No merge or deployment performed.
