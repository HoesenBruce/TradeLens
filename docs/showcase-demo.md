# Fictional TradeLens showcase v1

This is a synthetic practice portfolio, not broker evidence, company news, market
history or an investment recommendation. All account/news/plan names and import
filenames carry a fictional marker. Security codes are identifiers only; prices,
returns and catalysts are invented. No credentials, owner records, brokerage
exports or live price services are needed. Never connect this fixture to a real
portfolio or public production service.

## Run locally

Use three terminals from the repository root. Python 3.10+ and the repository's
Go toolchain are required. Choose a **new temporary directory** for every run;
keep the fixture running while viewing valuation charts.

```sh
python3 scripts/showcase-market.py --port 18964
```

```sh
SHOWCASE_DIR=$(mktemp -d)
cd api
TM_HTTP_PORT=8093 TM_DB_PATH="$SHOWCASE_DIR/showcase.db" \
TM_JWT_SECRET=fictional-showcase-local-secret-replace-before-deploy \
TM_MARKET_DATA_PROVIDER=http \
TM_MARKET_DATA_HTTP_BASE_URL=http://127.0.0.1:18964 \
    go run ./cmd/server
```

Create the first user **without an account**, then seed through authenticated APIs:

```sh
curl --fail-with-body http://127.0.0.1:8093/api/v1/setup \
  -H 'Content-Type: application/json' \
  -d '{"email":"fictional@example.com","password":"fictional-showcase-password"}'
python3 scripts/seed-demo.py --mode showcase \
  --api http://127.0.0.1:8093/api/v1 \
  --email fictional@example.com --password fictional-showcase-password
```

Web's development proxy defaults to port 8080. For browsing, run this disposable
API on port 8080 instead (stop any other API first), and change the seed URL to
match; then start Web with `make dev-web`. Select JPY,
the fictional account and September 1–8, 2026 for valuation. Screenshot/README
redesign is intentionally a separate issue. No Web code changes are included.

## Determinism and reset

Showcase v1 uses literal fixtures: September 1–8, 2026, with no RNG. `--seed`,
`--end-date` and `--account` overrides are rejected in showcase mode. The default
`legacy` mode retains the existing USD generator and its CLI behavior.

Showcase refuses any user with existing accounts or news before making writes.
It does not delete user data or resume a partial run. After an error or for a
second capture, stop the API and start it with a fresh temporary database,
then repeat setup. Accounts, imports, cash flows and news use public workflows;
there are no SQL writes. Random IDs and server-generated timestamps differ
between runs; quantities, prices, accounting results and immediate status counts
are reproducible. Do not use multiple seed processes against one user.

The underlying SBI importer independently deduplicates a repeated CSV. Whole-seed
rerun protection also prevents duplicate cash flows, plans and news.

## Expected results and audit

The SBI execution-history importer receives 12 fictional CSV rows and expands
現引/現渡 to 14 executions, forming seven closed trade groups:

| Case | Code | Net P&L (JPY) |
| --- | --- | ---: |
| Cash long | 6501 | 10,000 |
| Margin long | 285A | -5,000 |
| Margin short | 7203 | 10,000 |
| 現引 margin close / cash disposal | 6758 | 0 / 5,000 |
| 現渡 cash disposal / short settlement | 8306 | 5,000 / 0 |

Fees are explicitly zero. Deposit +1,000,000 and withdrawal -50,000 use the
cash ledger's signed-amount contract. All positions close: realized P&L 25,000,
contributed capital 950,000, cash and estimated account value 975,000. Six daily
valuation points use the existing account-value service and local Generic Bars
HTTP provider. The fixture serves only the six specified JP sessions and six
codes (including calendar proxy 1306); unknown symbols/intervals return 404 and
outside-window requests receive no invented extra history. Prices are explicitly
labelled `FICTIONAL-showcase-v1`, unadjusted synthetic bars. This is fixture
coverage, never evidence of historical market availability. Use the fixed range;
other ranges can be empty/incomplete.

6501 links to a separately saved long setup with target 1,200; the actual exit
comes from its 1,100 execution. This demonstrates planned versus actual price
without changing trade direction or accounting. Other trades carry fictional
journal notes. Existing shared accounting, SBI resolver, grouping and
valuation rules remain unchanged.

Three fictional news theses link numeric/alphanumeric assets to manual predictions
with 1/5-session horizons. Immediate performance is six horizon samples: four
pending, two unavailable, zero validated/incomplete, and null hit rate. The unknown
market on 7203 deliberately exercises the real unsupported-instrument evaluation;
285A validation exercises the actual awaiting-close path. No UI totals are seeded.

### Cases the supported API cannot immediately generate

Prediction create/update timestamps are owned by the server. Publication dates
are not prediction dates. Consequently a fresh seed cannot create historical
validated or incomplete price-evidence outcomes without bypassing that contract.
The script neither backdates database rows nor pretends the fictional September
prices cover newly saved predictions. Later validation requires genuine elapsed
eligible sessions plus an explicitly extended synthetic fixture. Current results
remain pending/unavailable. This is a documented showcase limitation.

Optional AI suggestions are omitted: no live provider configuration is touched,
no AI record is silently inserted/accepted, and no API key is used. Existing
`web/e2e/news-analysis.spec.ts` covers a fixture provider and explicit acceptance;
use that separate workflow when capturing the AI review UI.

## Verification

```sh
python3 scripts/test_showcase.py
# On another NEW API database with the fixture configuration above:
python3 scripts/test_showcase.py http://127.0.0.1:8093/api/v1
```

The second command performs setup itself. It checks all seven closed groups,
symbol preservation, exact P&L, six complete valuation points, ledger reconciliation,
actual performance counts, repeated-import dedup and pre-write seed refusal.
Local verification ran against two fresh SQLite databases. The legacy CLI was
also run on a separate disposable database with an empty-bars local HTTP fixture
for USD symbols (the JPY showcase provider intentionally rejects them). Privacy review is limited to the
changed fixture files: invented literals and reserved example email addresses,
with no external downloads or private input. No Web behavior changed; browser,
mobile, Postgres and remote CI were not validated for this infrastructure change.
