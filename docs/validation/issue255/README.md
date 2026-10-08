# Issue #255 presentation acceptance

Validated on 2026-10-08 against branch `docs/255-product-presentation`, based on
`38b9ee5b` (the #254 squash merge). No importer or accounting code changed.

## Executed checks

- `cd marketing && ./node_modules/.bin/eslint app lib`: passed.
- `cd marketing && ./node_modules/.bin/next build`: passed, including TypeScript
  and generation of 385 routes. Production server ran on port 4175.
- `git diff --check`: passed.
- Local README links, four Home translation key sets, and screenshot hashes were
  checked with Python assertions. All 26 original manifest images and five new
  byte-identical marketing copies matched their hashes/source bytes.
- `python3 scripts/test_showcase.py http://127.0.0.1:8097/api/v1`: passed against
  a fresh disposable SQLite API and `scripts/showcase-market.py` provider from
  this working tree. Read-back verified seven trades, JPY 25,000 net P&L,
  six valuation points ending at JPY 975,000, four pending/two unavailable
  predictions, 14-execution duplicate handling, and refusal to seed twice.
- The repository Compose stack with `.env.example` and pinned API/Web 0.2.1
  images started healthy in project `tradelens255`. Only host ports were changed
  to 18085/13005 for isolation. Web GET returned 200; fictional owner setup
  returned 201 and login returned 200. This was an empty disposable database.

## Browser acceptance

Used the Codex built-in browser against the production marketing build:

- Opened English, Japanese, Simplified Chinese and Traditional Chinese pages;
  inspected headings, copy and rendered screenshots.
- Switched dark → light → dark; reload retained dark. Screenshot assets and
  navigation icon loaded successfully.
- Reopened the language menu, cancelled with Escape without changing Japanese,
  then selected both Chinese variants and returned to English.
- Clicked Get started and reached the deployment section; returned home and
  opened Docs, where fork guide links and inherited-reference notice appeared.
- Inspected the SBI/news feature section and the original synthetic captures.
- Checked a 390 × 844 Web viewport: no horizontal overflow. Reset the viewport
  override afterward. This is responsive Web acceptance, not native mobile QA.
- Refreshed the final build after the footer wording change; confirmed a single
  main landmark and the optional-AI data-transfer notice.

The adjacent JPEGs are actual browser viewport captures. `marketing-en-dark.jpg`
was refreshed from the final build. Other captures record the exercised states;
subsequent changes only removed a nested main landmark and clarified footer
privacy wording. Product images are synthetic #254 assets, copied without edits;
source hashes remain in `docs/screenshots/screenshot-manifest.json`.

## Boundaries

No hosted site was published. Configure `NEXT_PUBLIC_SITE_URL` before publication.
No native mobile, PostgreSQL, source-build deployment fallback, live market-data
provider or AI integration was validated. Retained feature/comparison/reference
pages were built and labelled as inherited material; this is not a fresh audit
of every upstream claim or competitor detail. Remote CI is separate from these
local results; see the PR checks for its status.
