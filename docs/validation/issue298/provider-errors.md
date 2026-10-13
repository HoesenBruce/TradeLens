# Issue #298 provider diagnostic confidentiality

Development evidence, 2026-10-13. Independent security acceptance remains required. This change does not make Coach or the Phase 1 release ready.

## Source and scope

Base: `10c3be793691d029cb4fb1d8fe2dd401635f604a` (accepted #297 planning).
Selected source: blocked Draft #295 at `44c8761f4d236438d69a6c421ab4931e11aed893`.
Extraction follows [the A/C inventory](../../architecture/pr295-split-assessment.md): providererrors, OCR extraction/models, Flex client/outcomes, notification delivery/outcomes, OCR error response and Flex/alert DTOs, selected tests and the existing OCR E2E selector correction.

Additional small changes: safe debug formatting, explicit Flex handler/job formatting boundaries, sentinel-based polling exhaustion handling, and stronger provider/route/persistence tests. No dependency, migration, accounting, Coach output guard, successful News output change, historical-row rewrite or production setting change.

## Error flow and compatibility

- Provider exits format errors before API/service logging. Status strings and response bodies are untrusted; only numeric HTTP status is used.
- Flex's manual response and background job log format every failure, including downstream import errors. Persisted Flex/notification errors are formatted again at the write boundary.
- Every nonempty historical Flex/channel `last_error` returns the fixed generic message without modifying storage. Empty errors retain the existing DTO contract (Flex omits the field); successful status summaries stay intact.
- `errors.Is`/`As` retain cancellation, deadline, OCR busy/unavailable and Flex pending identity. Expo's dead-device sentinel still reaches dispatcher cleanup. Import, polling, retry and default/SBI accounting remain on their existing paths.
- `%v`, `%+v`, `%#v` and standard slog handlers format the safe message. The helper retains an internal cause; `Unwrap`/`As` are for identity/classification only. Never serialize, reflect, debug-dump or log the retrieved cause. OCR's timeout classifier reads `net.Error.Timeout`; Flex polls/checks pending; Expo dispatch checks the device sentinel. No changed production caller unwraps a cause into an output field.
- Both owner-only OCR and Coach connection/model-discovery callers use the shared `ocr.ListModels` boundary. Settings authorization remains owner-only. Coach generation/history and News successful output are outside this change's acceptance.

## Permanent checks

`api/internal/api/provider_failure_matrix_test.go`:

- Six diagnostic modes: HTTP status/body, embedded error, malformed response, transport URL, timeout, cancellation. OCR uses owner JWT, member JWT and member PAT; response bodies, headers and captured logs are checked.
- OCR/Coach settings test/models requests: owner JWT/PAT reach the mock; denied member JWT/PAT make no provider calls or durable changes. All six modes are exercised for both settings consumers.
- Manual and scheduled Flex, webhook delivery, persisted errors and old-row readback. Full unrelated-table hashes plus foreign status-row hashes; foreign requests make zero calls/changes. PAT usage metadata is normalized by the existing fixture.
- Alert evaluation can create its own events when delivery fails, as before; unrelated tables and other users' channel/Flex rows remain unchanged.
- Historical reads preserve all rows; empty/success statuses retain their contract.

`providererrors/errors_test.go` covers joined/wrapped causes, URL causes, `Is`/`As`, idempotence, nil, HTTP category, formatting and slog. Flex tests cover polling success/exhaustion, cancellation/deadline identity, six diagnostic modes, import success and sanitized stored outcomes. Notification tests cover status/transport/malformed/timeout/cancel, redirects, per-ticket errors, payload success and actual dispatcher dead-token cleanup.

Baseline reproduction: copied the API matrix into a disposable archive of the base main. Configuration, async delivery, stored readback and 15 of 18 OCR cases failed; the existing malformed OCR behavior already hid those three cases. Assertions print case IDs or row hashes, never provider bodies or bearer credentials. This is authored/adapted development regression evidence, not an independent review.

## Local validation ledger

- Full Go tests, vet and full race: final results recorded in the PR validation ledger.
- Web tests: 189 files / 1,091 tests passed.
- Web `vp check`: 0 errors, 347 existing warnings. TypeScript and production build passed.
- Diff whitespace: passed.

The built-in Codex browser used Web `:5228`, disposable SQLite/API `:8128`, and a loopback synthetic provider `:8428`; no vendor credentials or production database. OCR requests passed through the actual API and Go parser (no route interception): review, dismiss without applying, clean re-entry, confirm into form, save, list readback (8 shares at 12.34, fee 0.50), safe failure, successful retry, clear/reset and cancel. Rendered review/form/error states were inspected.

IBKR: seeded old error readback rendered generic; actual manual mock failure, toast, configuration readback and cancel rendered safe. Notifications: actual mock success, failure toast with numeric HTTP status, generic stored status, success retry and page-refresh readback. Rendered IBKR and notification states were inspected. Two existing view-transition AbortErrors occurred during rapid settings navigation; no diagnostic marker appeared in inspected browser logs. The API fixtures exercise provider error/log/DB boundaries independently of browser screenshots.

## Limits and delivery

Mock/local SQLite evidence only. Real vendors, PostgreSQL, deployed proxies and mobile are unverified; mobile is outside current fork scope. Successful provider payloads and owner-visible configuration fields are not audited here. Historical raw errors remain in storage/operator backups; only application DTO readback is masked, without destructive rewrites.

Keep the focused PR Draft until independent security review. No merge, Issue closure, release, tag, image publication, deployment or production configuration change is authorized by this implementation.
