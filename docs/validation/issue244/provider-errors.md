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
- Tests establish error-output boundaries, not immunity to all disclosure. Valid model completion content is intentionally preserved; a model putting a secret in a successful note is outside this error-remediation scope.
- Historical raw error rows are hidden on API read, not erased from operator-controlled databases/backups. Raw causes remain available internally for error identity; audited response/status/log consumers format only the safe error.
- The existing Coach browser mutation can produce an unhandled rejected promise on SSE failure. The browser test inspects it and permits only the exact sanitized message, then exercises successful regeneration. No Web product code or error-handling redesign is included.
- No #292/#293 branch modification, merge, Issue closure, release, publication or deployment is authorized here. #295 stays Draft pending independent security review.
