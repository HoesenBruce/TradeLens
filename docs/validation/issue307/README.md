# Issue #307 — SBI review direction

All records and prices here are synthetic. Web is the acceptance target; mobile
was not validated and is outside the current fork scope.

## Policy

Only explicit `sbi:cash`, `sbi:margin-long` and `sbi:margin-short` buckets opt in.
The bucket identifies acquisition/opening versus disposal/closing, independently
of account capabilities, planned direction and file order. Unknown/non-SBI
buckets retain the shared engine, including reversal fees and partial-close P&L.

The existing #267 `details.source_time_precision = "date"` marker is propagated
into accounting. No timestamp, occurrence identity or dedup key is rewritten.
For a Tokyo trading day containing a date-only record, reconcile all that day's
openings before its closes. This is a date-level inventory reconciliation, not
an intraday fill sequence or a reconstructed lot match. Prior supported inventory
carries across dates; a later day's purchase never repairs an earlier disposal.
Actual-time-only days retain timestamp order. Within each role, storage order
keeps the earliest supported opening's ID where possible.

Repeated uncertain openings/closings stay together until supported inventory is
exhausted. Opened quantity is cumulative, remaining quantity is nonnegative.
Explicit closes cannot open the opposite bucket. Orphan closes remain linked to
review records with zero supported opened/remaining quantity and null P&L; zero
entry is a storage sentinel, displayed as unavailable in Web. Excess disposals
consume only supported holdings and flag incomplete opening evidence. The source
quantity/price/fees remain intact. Known conversion and settlement reconciliation
continues through the existing strategies.

SBI cash uses its existing daily average acquisition-cost/rounding policy.
Margin keeps reported settlement P&L and its source; multiple openings without
settlement evidence have unavailable close P&L rather than an invented lot match.
Ordinary partial-close trade P&L remains null. A broker result on an unmatched
margin close remains in the source record and realized result, but cannot
establish a complete review trade's basis/P&L.

Date-only reviews have null exact holding duration and an explicit warning.
Web details show SBI source dates, suppress per-fill running positions and replay,
and place date-only markers on daily bars only. Horizontal execution price lines
still show the source prices. Actual-time records retain their normal presentation.
Account-value reconstruction and default non-SBI accounting are unchanged.

## Regression coverage

- All 24 file/ID permutations for date-only cash, margin-long and margin-short:
  entry 101, exit 111, cumulative opened 20, remaining 0; fees conserved and broker
  reported results retained. The dedicated cash reproduction is entry 100/exit 110,
  opened 10/remaining 0.
- Cash and margin missing/prior/insufficient opening evidence, later-date purchases,
  actual-time closing-first records, and short opening 70/cover 60 (remaining 10).
- CSV parse → SQLite commit → regroup → read-back; same-symbol cash/long/short
  isolation, repeat and reversed-file imports, stable read-back and review notes.
- Existing SBI cash daily cost, genbiki, genwatashi, settlement, split-boundary,
  account-value, importer, IBKR/options/default reversal and partial-close tests.
- Browser preview/back without writes, commit, list, drawer dismiss/re-entry,
  full-page detail/reload, execution tables, repeat/reversed import and chart
  interval/expanded-view transitions against a disposable real API.

## Historical recalculation and rollback

Deployment alone performs no bulk regroup, migration or historical reimport.
A new import or explicit account regroup uses the new strategy for that account.
Derived direction, prices, grouping, IDs/links and statistics may change.

Unchanged trade IDs retain their upserted notes and associations; automated tests
verify notes on stable IDs. A formerly closing-first short review may become a
long review keyed by an acquisition; multiple same-day reviews may merge. The
existing regroup prunes obsolete trade rows. It **does not migrate annotations**
from deleted IDs. Journals, plans, tags, attachments, share links and other linked
records must therefore be inventoried before a historical regroup. Do not assume
that stable imported execution IDs imply stable trade IDs.

1. Stop writes and take a consistent complete database backup (SQLite backup API
   or offline copy including WAL; PostgreSQL consistent dump), plus attachment files.
   Keep the currently deployed image/commit for rollback.
2. Restore into a disposable instance of this branch. Export the account's trade
   IDs, execution-ID links and all associated review metadata before regrouping.
3. With the required market provider working, call
   `POST /api/v1/trades/regroup` with `{"account_id":"<affected SBI account>"}`.
   Review quantities, warnings, settlement totals and the before/after execution-ID
   membership. Missing data remains a discrepancy; do not invent opening fills.
4. Map changed/merged IDs by their execution membership, and deliberately reattach
   journal/plan/tag/attachment metadata. Resolve conflicting reviews manually.
   Re-run regroup and verify stable IDs/results and the restored associations.
5. Repeat only the reviewed account procedure in production after a new backup.
   If results or associations are unacceptable, stop writes, restore the complete
   backup and attachment set, and restore the previous application version.

Simply downgrading code does not restore deleted trade associations. Do not run a
fleet-wide regroup or delete/reimport batches as an implicit deployment step.

## Reproduce browser acceptance

Start the synthetic bars provider:

```sh
python3 docs/validation/issue307/market-fixture.py
```

Run the API from `api/` with a **new disposable database**, a synthetic setup user,
`TM_HTTP_PORT=8091`, `TM_DB_PATH=/tmp/issue307-qa.db`, a disposable JWT secret,
`TM_CORS_ORIGINS=http://localhost:5173`, `TM_JOBS_ENABLED=false`,
`TM_ECON_CALENDAR_ENABLED=false`, `TM_MARKET_DATA_ENABLED=true`,
`TM_MARKET_DATA_PROVIDER=http`, and
`TM_MARKET_DATA_HTTP_BASE_URL=http://127.0.0.1:8093`. Start Web on port 5173.

```sh
cd web
E2E_API_URL=http://localhost:8091/api/v1 \
E2E_EMAIL=<synthetic-user> E2E_PASSWORD=<synthetic-password> \
E2E_BASE_URL=http://localhost:5173 \
node_modules/.bin/playwright test e2e/sbi-review-direction.spec.ts
```

The browser test uses the shared
`api/internal/importer/testdata/sbi-review-direction.csv` fixture. Each run creates
its own account and scopes Web to it. The bars provider serves fictional prices;
these screenshots demonstrate behavior, not production financial accuracy.

## Recorded validation

- Go: `go test ./...`, `go vet ./...`, and server build passed.
- Web: `vp check` passed with 0 errors (347 existing warnings); TypeScript build
  check passed; 191 test files / 1,094 tests passed.
- Chromium: `sbi-review-direction.spec.ts` passed against the current compiled
  API, disposable SQLite database and synthetic HTTP bars provider. The built-in
  browser tool was unavailable, so Playwright was used.
- Screenshots inspected: [list](list.png), [daily chart](daily-chart.png),
  [execution details](margin-executions.png), [partial short](partial-short.png),
  and [missing opening](missing-opening.png).
