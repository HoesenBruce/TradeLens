# Provider error boundary — PR #295

Baseline main: `25fe2f97d41053f2eb8e84362e28ad950c35f55e`. Previous #295 head: `028a47faf78dfd7c20ff11050c432dc880c79f48`. This supplements the route inventory: resource ownership alone does not secure provider error text.

## Boundary matrix

All API paths require JWT or PAT. Ordinary private resources remain scoped to their authenticated owner; OCR/Coach configuration management additionally requires instance-owner authorization.

| Path / service | Upstream source and scope | Prior propagation | Fix / dynamic evidence |
|---|---|---|---|
| `/ocr/parse`, `ocr.ExtractTradeFromImageIn` | Instance vision credentials/prompt; HTTP, transport, decoded API or malformed output errors | Generic response from earlier #295, but raw provider errors could still enter logs | Preserve generic 502, timeout 504, unavailable/busy 503. Sanitize the service error before callers/loggers; JWT/PAT/owner failure matrix, existing OCR regression and timeout/retry tests |
| `/trades/:id/coach`, `coach.GenerateReview` | Owned trade, shared instance AI configuration | Raw error in HTTP 200 `{source:error,error}` and logs | Stable `coach generation failed`, existing response shape/status; service error sanitization. JWT/PAT/owner auth, embedded error, malformed output, timeout, cancel and format fallback |
| `/trades/:id/coach/stream`, `coach.GenerateReviewStream` | Same scope, SSE frames | Raw error in `event:error`, transport/parser diagnostics in logs | Stable sanitized error event; preserve note/done framing, timeout/cancel/format fallback identity. Failure before first note, safe partial note followed by upstream error, malformed stream and successful existing stream tests; no failed review persisted |
| `/news/:id/analyze`, `coach.AnalyzeNews` | Owned news, instance AI configuration | Client messages already generic; underlying shared client error could contain provider content | Keep existing generic 502/504/503. Sanitize returned service errors; JWT/PAT/owner matrix and unchanged-news read-back |
| `/settings/{ocr,coach}/{test,models}`, `ocr.ListModels` | Instance-owner-only diagnostics | Raw models errors returned even to owner | Only safe HTTP status/category returned. Members/PAT denied before provider call; owner mock failures triggered and logs/responses scanned |
| `/accounts/:id/flex-sync/run`, `flexsync.Client.FetchStatement` | Per-user account token/query; URL query, XML error, transport failure | Raw response, settings `last_error`, scheduled job logs; token hint masking did not protect error URLs | Sanitize provider failure at client boundary. Keep statement-pending identity and local CSV-layout guidance. JWT/PAT transport failures, mock XML failure through actual scheduled job, scoped SQL/read-back and successful import/dedup regressions |
| `/settings/alert-channels/:id/test`, webhook sending | Per-user channel, credential-bearing target URL; blocked redirect/transport error | Raw response and `alert_channels.last_error`/logs | Sanitize delivery errors, store safe status; real mock redirect failure, JWT/PAT direct API and asynchronous risk-alert dispatch; response/log/SQL scans |
| Expo delivery | Per-user push token; ticket error or transport failure | Raw ticket text persisted/logged by dispatch | Sanitize global/per-ticket errors while preserving `DeviceNotRegistered` identity and token disabling; ticket mock, webhook/Expo existing successful sends |
| User-readable historical `last_error` | Existing Flex/channel status rows | Old raw diagnostics readable after code fix | Return generic text for nonempty legacy error fields; seeded legacy rows checked through API. No history/database rewrite |
| `providererrors.Safe` / `HTTPStatus` | Shared error formatting boundary | Provider strings, URLs, headers and prompts were treated as trusted messages | Safe status/cancellation/timeout/category only; retain `errors.Is`/`errors.As` behavior for existing control flow. Format and slog tests contain no sentinels |

## Regression design

`provider_failure_matrix_test.go` reuses #244's two-user SQLite fixture and complete-table digests. It asserts that failures actually reach the mock, never persist a failed review or news update, and remain absent from follow-up APIs and captured request/service logs. Connection/channel outcome writes are intentional and checked separately; foreign requests remain denied.

Sentinels represent owner API keys, Bearer/Authorization values, private prompts, IBKR credentials and webhook tokens. Tests cover JWT, PAT and the trusted instance owner. Success paths retain existing OCR extraction, model discovery, Coach JSON/SSE, news review, IBKR import/dedup, notifications and accounting regressions.

New API regressions fail on the previous #295 production code with only the tests added. The earlier tests asserting raw provider text now assert exact sanitized text and absence of the original text. No assertions, timeout limits or successful-flow cases were removed. Local CSV layout guidance remains intact.

Existing browser fixtures are aligned with the current login form and title-cased News fields. OCR now exercises the existing `Fill form` confirmation before asserting prefilled values. News retains the original “No data was changed” assertion against the actual API response and adds the current generic UI error assertion; its before/after data equality remains.

## Commands

From `api/`:

```sh
go test ./...
go vet ./...
go test -race ./...
go test ./internal/api ./internal/alerts ./internal/providererrors -run 'TestCrossUserIsolation|TestProviderExpo|TestProviderWebhook|TestSafe' -count=3
```

From `web/`, after starting a disposable API at :8098 and Vite at :5195:

```sh
./node_modules/.bin/vp test run --maxWorkers=2
./node_modules/.bin/tsc -b --noEmit
./node_modules/.bin/vp check
./node_modules/.bin/vp build
E2E_API_URL=http://localhost:8098/api/v1 E2E_BASE_URL=http://localhost:5195 PLAYWRIGHT_BROWSERS_PATH=/private/tmp/tradelens-phase1-pw ./node_modules/.bin/playwright test e2e/provider-errors.spec.ts --timeout=120000 --reporter=json
E2E_EMAIL=owner295@example.invalid E2E_PASSWORD=Synthetic295-acceptance-password E2E_API_URL=http://localhost:8098/api/v1 E2E_WEB_URL=http://localhost:5195 E2E_BASE_URL=http://localhost:5195 PLAYWRIGHT_BROWSERS_PATH=/private/tmp/tradelens-phase1-pw ./node_modules/.bin/playwright test e2e/ocr.spec.ts e2e/news-analysis.spec.ts --timeout=120000 --reporter=json
```

From root: `git diff --check`. Exact final head, outcomes and CI URLs are recorded in the updated Draft PR #295 description.

## Limits

- SQLite/local storage and synthetic providers are exercised. PostgreSQL, alternative storage, real IBKR/AI/Expo/webhook services and deployed proxies are unverified.
- Tests establish the documented boundaries, not immunity to all disclosure. Coach provisional and successful output now has the bounded content checks below. Arbitrary successful OCR/News model content is outside this targeted Coach remediation.
- Historical raw error rows are hidden on API read, not erased from operator-controlled databases/backups. Raw causes remain available internally for error identity; audited response/status/log consumers format only the safe error.
- The existing Coach browser mutation can produce an unhandled rejected promise on SSE failure. The browser test inspects it and permits only the exact sanitized message, then exercises successful regeneration. No Web product code or error-handling redesign is included.
- No #292/#293 branch modification, merge, Issue closure, release, publication or deployment is authorized here. #295 stays Draft pending independent security review.

## Coach pre-error note remediation

Vulnerable baseline: `b44857381d403b8885e4fdca0626f65598ea7ca4`. The [independent blocking review](https://github.com/HoesenBruce/TradeLens/pull/295#issuecomment-6137572036) reproduced owner API key/private prompt disclosure with member JWT/PAT. A complete provisional note was emitted before the later provider error. The exact `TestIndependentCoachPartialFailureMustNotDisclose` fixture was first run on this baseline and failed for both identities. After remediation it additionally requires **only one generic error event, with no unsafe note**; the provider/configuration/authentication and secret-absence assertions remain. An assertion requiring the vulnerable note event is not an acceptance requirement.

### Data flow and changed boundaries

1. `GenerateReview` / `GenerateReviewStream` construct messages from the authenticated owner's trade and instance Coach instructions. API key remains in the transport Authorization header. Known transport keys, URL credentials and credential-header lines accidentally embedded in instructions/journal context are omitted from model-visible messages; necessary private instructions otherwise remain.
2. Provider frames are decoded and accumulated across chunk boundaries. A **complete note** is the smallest output unit. `outputGuard` checks decoded headline/detail and their combination **before** invoking the note callback; the API handler can flush only validated notes. It rejects the entire generation rather than emitting a partially redacted credential.
3. The final review/next action passes the same guard before JSON, SSE `done` or persistence. Blocking format fallback is guarded too. Unsafe content is rejected on success as well as failure. Safe previews already sent retain their normal ordering; an unsafe later note produces a generic error and no saved review.
4. Owned historical Coach reviews are checked on read against current known configuration. Unsafe rows are omitted without rewriting the database; the log contains only their review ID. Old unknown/rotated secrets cannot be identified reliably without an operator history audit.

### Bounded recognition contract

The guard rejects current known API keys, URL user/password and named credential query values, identifiable custom-prompt echoes (including the credential-minimized prompt), and recognizable Authorization/Bearer/Basic/credential diagnostic text. It compares normalized decoded fields and recognizes whitespace/punctuation splitting, URL encoding, HTML entities, JSON Unicode escapes, Base64/Base64URL and hex representations. JSON scalar values and property names are also checked in source order, without intervening labels, so short pieces inside one structured note cannot bypass checks. This case independently failed on the first candidate `41edcd7a` before that additional boundary was fixed. Known long keys and prompt text also have conservative fragment checks (12 and 32 normalized bytes respectively); short keys require whole-token matches to avoid interpreting the letter `k` in ordinary words as a credential.

This is a **bounded recognizable-echo guard**, not a proof that arbitrary free-form model text is confidential. Novel ciphers, semantic paraphrases, steganography or arbitrary fragments rearranged across unrelated notes are not generally identifiable. If that stronger guarantee is required, use a trusted structured provider returning validated template IDs and facts drawn from the owned trade, then render server-controlled text; do not claim an unrestricted free-form stream can satisfy it. That product design is proposed, not implemented by this security patch. Conservative matching can reject legitimate text that quotes private instructions; it fails closed and allows a retry, never stores the rejected review.

Bounds: 64 KiB cumulative decoded review content, 4 MiB provider wire stream, at most five emitted notes; encoding inspection has three decoding levels, JSON structure inspection has 16 nesting levels, and checks have 128 inspected views/known values and a 256 KiB inspection budget per check. Oversized, invalid UTF-8, malformed SSE or over-budget output fails closed. Strings that are not valid supported encodings remain subject to plaintext checks; this does not promise detection of arbitrary encodings. Empty SSE heartbeat frames are accepted. Existing request cancellation, timeout status/error identity, event envelope and safe JSON/stream success behavior are preserved.

### Added evidence and commands

`coach_output_security_test.go` exercises real routes with stored synthetic owner settings and a mock that verifies Authorization/private instruction delivery, member config denial, complete response/event/header/log scans, and complete-table hashes. Cases include unsafe note followed by failure **and success**, safe success/failure, five-byte chunk splitting, structured/URL/Unicode/HTML/Base64/hex/multiline echoes, split fields, mixed notes, unsafe next action, empty/malformed frames, oversized content, foreign trade denial and historical readback. `TestIndependentCoachActualConnectionTermination` uses real HTTP client disconnect, provider disconnect and stalled-stream timeout under JWT/PAT. `output_stream_test.go` withholds completion until a safe note actually arrives, verifies final two-note success, wire/content bounds and unsafe blocking fallback. Existing tests remain; safe streaming API order is `note`, `note`, `done`.

```sh
# api/: baseline reproduction before production edits (expected FAIL), then fixed PASS
go test ./internal/api -run TestIndependentCoachPartialFailureMustNotDisclose -count=1
go test ./internal/api ./internal/coach ./internal/alerts ./internal/providererrors -run 'TestCrossUserIsolation|TestIndependentCoach|TestCoach|TestProviderExpo|TestProviderWebhook|TestSafe' -count=3
go test ./... -count=1
go vet ./...
go test -race ./... -count=1

# web/
./node_modules/.bin/vp test run --maxWorkers=2
./node_modules/.bin/tsc -b --noEmit
./node_modules/.bin/vp check
./node_modules/.bin/vp build
E2E_API_URL=http://localhost:8102/api/v1 E2E_BASE_URL=http://localhost:5199 PLAYWRIGHT_BROWSERS_PATH=/private/tmp/tradelens-phase1-pw ./node_modules/.bin/playwright test e2e/provider-errors.spec.ts --timeout=120000 --reporter=json

# root
git diff --check origin/main...HEAD
```

The Chromium test uses the actual API and a real mock provider: a credential-bearing provisional note split across frames is rejected, then retry displays a safe note while the provider deliberately withholds final completion. It captures page/console errors and failure/incremental/success screenshots. The inherited sanitized rejected promise remains a separate UI limitation. Exact final head, command outcomes and **new-head** required CI URLs belong in #295's updated evidence; previous-head CI is not acceptance evidence. No merge, Issue closure, production database change, release or deployment is part of this task.
