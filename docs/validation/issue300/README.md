# Issue #300 design review evidence

Scope: documentation and a disposable synthetic schema illustration, based on
main `10c3be793691d029cb4fb1d8fe2dd401635f604a`. No production implementation,
provider request, real secret, DB mutation, browser change or operational action.
[Contract and sign-off requirements](../../architecture/coach-output-contract-v1.md).

## Reproducible local check

```sh
python3 docs/validation/issue300/contract_sketch.py
```

Result on 2026-10-13: **8 test methods PASS**, including parameterized adversarial
cases. This is a new reproducible illustration, not a rerun of the historical
16-test `/private/tmp/tradelens-295-poc/poc.py` cited by the original planning.
The sketch is not imported by the application or a production validation library.
It emits catalog IDs to show provenance; it does not implement the renderer.

| Test | Evidence supplied |
|---|---|
| all_templates | each of four templates can select a valid synthetic typed fact |
| schema_attacks | every missing/null key, extra prose/model key, mismatched enum, arbitrary params, wrong ref shape and version type rejected |
| decoder_attacks | duplicate decoded keys including Unicode alias, duplicate nested keys, invalid UTF-8/surrogate, partial/trailing JSON, depth/record budget, NaN and exponent version rejected |
| reference_binding | foreign user/account/trade/run, unknown ref, missing value, bool/int confusion and wrong fact type rejected |
| numeric_domains | out-of-range grade/streak and bool/float substitutes rejected |
| incremental_and_late_failure | iterator yields a safe catalog choice while end is held; late malformed/duplicate/after-end data never completes |
| terminal_and_save | empty valid completion; missing normal finish or simulated failed save rejects |
| contradictory_snapshot | execution grade and absent-grade selections cannot coexist |

Not implemented by the sketch: blocking envelope/fallback, quotas, wire/decompression
limits, pre-allocation depth guards (Python parses first), real UTF-8 stream chunk
assembly, localized rendering, full fact resolver, DB readback, SSE/client schema,
real persistence, lease fencing, timeouts/revocation/concurrency, old clients,
cache invalidation or proxy flushing. The fake save flag is not transaction proof.
Those require permanent Go/Web tests and independent D2 acceptance in #301/#303.
No full Go/Web tests or mobile/browser E2E are needed for this docs-only change.

## Author adversarial review: escape-field ledger

This is an author review, **not independent security approval**.

| Candidate escape | Decision / remaining proof |
|---|---|
| `headline`, `detail`, `model`, usage, error text | absent from model schema; public text is literal catalog or fixed status text; C must inspect every response/log sink |
| `params`, arbitrary numeric digits | empty object only; no display interpolation, money, confidence or percentage in catalog 1 |
| `fact_refs` as global ID/path | allocated request-local registry only; ownership/account/trade/run binding checked, model cannot dereference DB |
| `category`, `priority`, `action` as extra choice | exact metadata equality with template; completion action uses fixed precedence |
| journal/tag/symbol/private prompt injection | excluded from input DTO, registry and renderer; no whole-struct marshaling |
| ambiguous risk/grade fact | stop-price presence is distinct from initial risk amount; review grade uses existing TradeQuality |
| failed lookup treated as missing | unknown fact is ineligible; only successful lookup may establish known absence |
| forged stored snapshot | validate shape/types/bindings/predicates on read; corrupt/unknown rows withheld; compromised DB/host remains outside threat model |
| partial/late provider failure | validate before each emission; end plus normal transport completion required; no completed write on bad output |
| fallback downgrade | one classified compatibility fallback before any finding, same schema/budgets; malformed output cannot trigger fallback |
| save failure / lost completed event | no transient success; durable unique key and fenced reservation reconcile ambiguous commit without a repeated provider call |
| stale owner/PAT/account binding | authoritative authorization before emission/readback/commit; serialized disable/revocation requires real race proof |
| legacy model/prose / rotated secret | fixed withheld placeholder, no secret recognizer or raw download route; preserve rows unchanged |
| old client / rollback | generation explicitly unavailable for absent version; safe history adapter; rollback retains E containment and v2 tables |
| News successful prose | no approval inherited from Coach or sanitized errors; independent Core containment/acceptance |
| provider choice/order/timing | explicitly residual, not eliminated; security/product reviewer must accept this limitation |

## Change reach and review status

Production inspection covered current `coach/context.go`, `execution.go`,
`session.go`, API `coach_handlers.go` and route registrations, analytics
`RiskRecorded` semantics, and Web `trades.coach.types.ts`. These establish current
fact/source and client compatibility differences; they do not constitute an
exhaustive bypass audit. The implementation owner must cover the migration
plan's full inventory: generation/stream/prompt callers, API/history/settings
capabilities, SQLite/PostgreSQL/sqlc, Web API/parser/hooks/translations/caches,
fixtures and direct-service/News paths. Exports/shares require explicit inventory
in C/E; this review does not claim their absence.

Local checks: sketch above, relative Markdown link existence, catalog coverage
(four templates × five locales), and `git diff --check`. No production file changed.

Pending: named boundary owners, independent security review at final PR SHA,
product acceptance of reduced personalization/old-client availability and residual
channels, five-locale wording acceptance. **B_APPROVED=NO**. Keep the delivery PR
Draft and #300 open until these are recorded. C/#301 stays blocked by B; this has
no bearing on independent Core D1 readiness.
