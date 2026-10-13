# PR #295 security split assessment

Status: technical assessment preserved from #296; release scheduling is superseded by the two-track roadmap. Architecture C; #295 remains blocked Draft. Reviewed 2026-10-13.

## Verified baseline and evidence

- Repository: `HoesenBruce/TradeLens` (explicit `-R` used for GitHub queries).
- Fetched `origin/main`: `25fe2f97d41053f2eb8e84362e28ad950c35f55e`.
- Fetched #295: `44c8761f4d236438d69a6c421ab4931e11aed893`, OPEN/Draft, nine commits, 39 changed files against current main. Both supplied SHAs match.
- #294 MERGED 2026-10-12, merge commit equals current main.
- #292 OPEN/Draft, actual head `a5577e7ebc864b36ae46af1ca8a155af47f0819b`; its body still describes an older `489d2b1a…` head. Do not reuse that body's CI as current integration evidence.
- #293 OPEN, not Draft, head `ef785d075afc9810761fddb22ba3e9954ae5b912`; release readiness remains blocked.
- [Latest independent finding](https://github.com/HoesenBruce/TradeLens/pull/295#issuecomment-6145818755) supersedes conditional recommendation B in #295's body/docs. Earlier CI and browser passes do not exercise the new ordered cross-note oracle.
- `stream.go` invokes `guard.note` before each callback; `sseEvent` immediately flushes. `guard.review` concatenates notes only at final completion. Final rejection and no persistence do not prevent earlier disclosure. No downgrade is justified.

Workspace preflight found the owner's checkout clean on `feat/272-sqlite-backups` at `a6511672`; unrelated existing worktrees, including prunable entries and independent review checkouts, were preserved. Documentation uses separate `docs/244-pr295-security-architecture` from current main at `/private/tmp/tradelens-295-design`. No existing PR branch changed; no implementation branch created.

## Decision and extraction method

`PR_295_SPLIT_FEASIBILITY=PARTIAL`: independent provider remediation is EXTRACTABLE, but #295 cannot safely merge as a whole. Category A is conditional on shared-helper and end-to-end error-flow validation, not on its file label.

Prefer one small **Issue A provider PR**, including the 42-line helper, OCR, IBKR, notification write/log/readback fixes, and the selected tests. Land write and historical readback protection together. A separate foundation PR adds sequencing overhead without an independent consumer; use it only if reviewers cannot audit the combined provider diff. Internal review order: helper → OCR → IBKR/notification → persisted readback. Coach changes remain excluded.

| Method | Assessment |
|---|---|
| Selected commit cherry-pick | First commit `028a47fa` is OCR-focused but handler-only logging protection is incomplete. `1f28f595` mixes all providers and Coach; later production commits are Coach-specific. Never cherry-pick all nine. First commit is usable only after reviewing its docs/assertions and adding source sanitization. |
| Selected final file/hunk diff | Recommended: take A files and helper from the exact reviewed head; split mixed matrix/doc files by test/function; record source SHA and selected paths in new PR. Validate no dependency on outputGuard. |
| Minimal reimplementation | Acceptable for mixed tests/helper boundaries and stale docs; require side-by-side reviewed diff and the same adversarial fixtures. Avoid gratuitous rewrites. |
| Foundation PR | Optional only; helper must contain no prompt recognition, encoding guard or Coach-specific policy. No release acceptance from a helper alone. |

Coach's generic error/HTTP-status hunks are technically independently useful, but extracting them adds mixed-function review and does not fix successful output. The minimal A scope defers them to containment/Coach work, rather than implying Coach is secured.

`Safe` is a formatting boundary, not irreversible scrubbing: `Unwrap` retains the raw cause to preserve `errors.Is`/`As`. Current Expo dispatch uses `errors.Is` for dead-token cleanup, and Flex poll exhaustion explicitly preserves `ErrStatementPending`. Audit every caller/log/DTO/job: prohibit raw-cause serialization, reflection/debug dumps and arbitrary upstream text in HTTP status. Standard logging and `%v`/`%+v` are covered by existing tests; `%#v`, custom handlers and future unwrap callers need explicit policy/tests. Consider typed classification with no raw cause only if callers can retain needed sentinel behavior with a small implementation. No input-sensitive text matching belongs in this helper.

## Complete file inventory

A = independent provider work; B = Coach-specific track; C = shared/mixed dependency. “Yes” means eligible after the stated acceptance tests on a fresh extraction head, not already accepted for merge. Test requirements below include existing callers, not just newly added tests.

| Path | Category | Reason | Dependencies | Safe to extract | Required tests | Migration / compatibility |
|---|---|---|---|---|---|---|
| `api/internal/alerts/notify.go` | A | Webhook HTTP status and Expo batch/ticket diagnostics sanitized | providererrors; dispatcher errors.Is | Yes, with helper | Redirect/transport/status/malformed tickets; DeviceNotRegistered cleanup; success | Error text becomes generic; token eviction identity preserved |
| `api/internal/alerts/notify_test.go` | A | Webhook redirect and Expo ticket sentinel tests | notify.go; existing testService | Yes | Run existing and added tests; add dispatcher eviction check | Test only; no migration |
| `api/internal/alerts/service.go` | A | Sanitize persisted outcome and delivery log | providererrors; notify | Yes | Background delivery, log capture, scoped status writes | New last_error generic; no old-row rewrite |
| `api/internal/api/alert_handlers.go` | A | Mask every nonempty historical last_error in DTO | Existing toAlertChannelDTO callers | Yes, together with write boundary | List/create/update/test readback; empty error stays empty | DTO shape unchanged; diagnostic specificity reduced |
| `api/internal/api/coach_base64_routing_test.go` | B | JWT/PAT folded encodings, successful output and legacy history | Coach guard/stream and associated fixtures | No; retain/adapt on C track | LF/CRLF JSON/plain; success/failure/fallback; DB hashes | Test only; old guard internals not a v2 requirement |
| `api/internal/api/coach_concurrent_security_test.go` | B | Concurrent users, failed attempt then incremental retry | Coach guard/stream and associated fixtures | No; retain/adapt on C track | Real HTTP first finding before release; per-user persistence; race | Test only; old guard internals not a v2 requirement |
| `api/internal/api/coach_handlers.go` | B | Generic error plus current-config history filtering | providererrors; ValidateReviewOutput; store | No whole file; generic-error hunk independently possible but deferred | Blocking/fallback/history/metadata/ownership/persistence | History currently silently skips rows; C replaces with explicit placeholders |
| `api/internal/api/coach_output_security_test.go` | B | Plaintext/encodings/metadata/history/disconnect routes | Coach guard/stream and associated fixtures | No; retain/adapt on C track | All cases plus published ordered cross-note oracle | Test only; old guard internals not a v2 requirement |
| `api/internal/api/coach_provenance_routing_test.go` | B | Malformed decoded structured representations | Coach guard/stream and associated fixtures | No; retain/adapt on C track | JWT/PAT failure/success/fallback/history; safe logs | Test only; old guard internals not a v2 requirement |
| `api/internal/api/coach_stream_handler.go` | B | Generic final SSE error only | GenerateReviewStream; sseEvent | No in minimal A | Pre-error event transcript, flush order, cancel | Does not retract earlier note events |
| `api/internal/api/cross_user_isolation_test.go` | A | Adds owner-config OCR JWT/PAT regression | main isolation fixtures; OCR tests | Yes, added hunks only | Provider reached, settings 403, 502 generic; redact assertion diagnostics | No migration; do not copy credential-printing failure assertions |
| `api/internal/api/flex_sync_handlers.go` | A | Hide persisted IBKR diagnostics on read | Existing toFlexSyncDTO callers | Yes, with write boundary | Own account/list and foreign account JWT/PAT | Same DTO; nonempty last_error becomes generic |
| `api/internal/api/ocr_handlers.go` | A | Generic OCR 502 response | OCR service errors.Is; source sanitization for logs | Yes; handler alone does not sanitize logs | 502 redaction, 503 unavailable, 504 timeout, successful parse | Status/code stable; message intentionally less detailed |
| `api/internal/api/ocr_handlers_test.go` | A | Replace upstream reflection assertion | OCR handler | Yes | Updated redaction plus existing busy/unavailable cases | Test only |
| `api/internal/api/provider_error_boundary_test.go` | B | Coach blocking/SSE owner-config diagnostic test | Coach transport and routes; isolation helpers | No in minimal A; retain for C | JWT/PAT error paths, DB unchanged | No production effect; useful regression retained |
| `api/internal/api/provider_failure_matrix_test.go` | C | Mixed OCR/Coach/News/config/async/readback matrix | isolation fixtures; sseFrames; noProviderSecrets | Selected subtests/hunks | A keeps OCR/settings/async/stored errors; B/C retains Coach/News output cases | Split helpers locally; no free-text streaming acceptance in A |
| `api/internal/coach/base64_security_test.go` | B | Folded encoding and finite guard budgets | Coach guard/stream and associated fixtures | No; retain/adapt on C track | Retain attack corpus; do not require v2 to accept free-text Base64 | Test only; old guard internals not a v2 requirement |
| `api/internal/coach/generate.go` | B | Mixed transport sanitization, prompt replacement, output guard | providererrors; output.go; News requestContent caller | No whole file; transport-only hunks technically separable | JSON/fallback/News/error identities; schema C replaces output path | Shared requestContent impacts News; arbitrary text remains unsafe |
| `api/internal/coach/news.go` | B | Sanitize AnalyzeNews errors, not successful arbitrary strings | providererrors; requestContent | No in minimal A; include containment/Coach scope | News success/error/explicit accept; no auto persistence | Uses shared Coach credential; separate successful-output contract needed |
| `api/internal/coach/output.go` | B | 470-line recognizable-text/encoding guard | VisionConfig; private prompt/transport recognition | No | Historical P1 fixtures retained as boundary tests | Not a common sanitizer; rotated secrets and indirect channels unproven |
| `api/internal/coach/output_stream_test.go` | B | Incremental completion, wire bounds, fallback | Coach guard/stream and associated fixtures | No; retain/adapt on C track | Adapt to structured findings; prove before-completion emission | Test only; old guard internals not a v2 requirement |
| `api/internal/coach/output_test.go` | B | Recognizable echoes and message credential removal | Coach guard/stream and associated fixtures | No; retain/adapt on C track | Retain disclosure scenarios; replace permissive text success expectations | Test only; old guard internals not a v2 requirement |
| `api/internal/coach/provenance_security_test.go` | B | Strict origin/decoded views and bounds | Coach guard/stream and associated fixtures | No; retain/adapt on C track | Keep historical oracle; v2 rejects arbitrary fields before rendering | Test only; old guard internals not a v2 requirement |
| `api/internal/coach/stream.go` | B | Per-note guard, budgets, final aggregate guard | output.go; providererrors; callback | No | Cross-note pre-flush regression; fallback; real incremental/cancel/retry | Known P1 remains; never promote current callback placement |
| `api/internal/coach/stream_test.go` | B | Existing callback test adapted to guard signature | Coach guard/stream and associated fixtures | No; retain/adapt on C track | Retain once-only streaming semantics under v2 | Test only; old guard internals not a v2 requirement |
| `api/internal/flexsync/client.go` | A | Sanitize two-step provider errors; retain pending identity | providererrors; ErrStatementPending retry callers | Yes | Expired token, URL/transport, poll exhaustion, timeout/cancel, successful CSV | Safe wrapper changes text/type; errors.Is pending preserved |
| `api/internal/flexsync/flexsync_test.go` | A | Generic error and stored outcome expectations | client.go; sync.go | Yes | Full Flex sync tests including accounting/import success | Test only |
| `api/internal/flexsync/sync.go` | A | Sanitize RecordOutcome persisted errors | providererrors; background job | Yes | Direct and scheduled runs; scoped DB hashes; success summary | No historical rewrite |
| `api/internal/ocr/models.go` | A | Sanitize model-list transport/status errors | providererrors; OCR and Coach admin settings callers | Yes | Both settings test/models routes; member denied; owner safe errors | Error text only; successful model IDs remain separate owner-only data |
| `api/internal/ocr/vision.go` | A | Sanitize extraction exit/status failures | providererrors; OCR retry/timeout sentinels | Yes | Malformed/body/status/transport/retry; valid image parse and confirmation | No parser/accounting semantic change |
| `api/internal/providererrors/errors.go` | C | Safe formatting with internal cause retention | Go errors/net/context only; no Coach guard | Yes, foundation inside Issue A | nil/idempotence/wrapped HTTP/cancel/timeout/Is/As/log formatting | Unwrap can expose cause if misused; never serialize/debug-print cause |
| `api/internal/providererrors/errors_test.go` | C | Safe formatting and sentinel identity | providererrors | Yes; extend | Existing tests plus ErrorIs/As consumers, %+v/%#v policy, joined errors | Test only; does not prove all log sinks safe |
| `docs/validation/issue244/README.md` | C | Mixed provider scope and Phase 1 status | Verified A test results and current review verdict | Rewrite selected evidence | Links/head binding/coverage recount | Do not inherit stale readiness claims |
| `docs/validation/issue244/coach-structured-output-proposal.md` | B | Short predecessor to architecture C | Design and migration decisions | No need; superseded by this design | Design review against full threat model | Proposal only |
| `docs/validation/issue244/endpoints.md` | C | OCR route reclassification and totals | OCR JWT/PAT regression; main route inventory | Yes after A passes | Inventory test; distinguish tested from complete coverage | Documentation only |
| `docs/validation/issue244/provider-errors.md` | C | Mixed provider ledger and obsolete conditional recommendation B | All track evidence; latest independent comment | Rewrite; no wholesale copy | Separate local/remote/independent evidence by SHA | Superseded by confirmed C finding; preserve historical provenance |
| `web/e2e/news-analysis.spec.ts` | B | Selector/confirmation and safe failure expectations | News endpoint; real disposable API | Defer to Coach/News track | Success/review/accept/error/retry; no mutation on failure | Test only; selector fixes not security proof |
| `web/e2e/ocr.spec.ts` | A | Correct login selector and explicit Fill form step | OCR UI baseline | Yes | Review then confirm/save; cancel/retry/readback as applicable | Test only; mocked UI test is not real-provider boundary proof |
| `web/e2e/provider-errors.spec.ts` | B | Despite name, real Coach browser/route fixture | Coach settings, mock provider, JWT/PAT | No | Replace free-text success with v2; retain error and incremental timing | Test only; currently permits sanitized rejected Promise |

## Audit limits and evidence from this planning turn

Static inspection covered all changed production files and tests, full nine-commit/file inventory, relevant current callers, Coach context/history/storage, Web types/stream hook, and latest independent review. This is not a fresh comprehensive penetration test.

The published independent unit was copied verbatim from the linked comment into a disposable archive of #295's exact head at `/private/tmp/tradelens-295-repro/api/internal/coach/planning_repro_test.go` (not a shared checkout). `go test ./internal/coach -run '^TestFinalIndependentAggregateBeforeCallback$' -count=1` **FAIL** (exit 1; package 0.936s): ordered disclosure and unsafe callback approval both reproduced. No production file was edited. The comment's full JWT/PAT/race/Web results are independently reported historical evidence, not rerun results of this planning task.

A disposable structured-boundary sketch at `/private/tmp/tradelens-295-poc/poc.py` passed 16 Python unit assertions (schema/version/unknown field/template/duplicate/partial/budget, owner/account/trade/reference checks, safe rendering, incremental success/failure). It is a design illustration, not a production parser, HTTP/SSE integration, persistence or Go race proof. No raw model payload or credential value is part of delivery.

Release and acceptance decisions: [roadmap](phase1-security-roadmap.md). Contract: [architecture C](coach-secure-output-design.md). Storage and rollout: [migration](coach-security-migration-plan.md).

## Product scheduling clarification

AI Coach is retained; Category B remains blocked from extraction and requires its own B/C/D2 recovery. Category A plus independently verified E isolation enables Core D1 assessment without waiting for B, C or D2. No unsafe legacy enablement bypass is accepted. News analysis has an independent Core confidentiality acceptance path, not a dependency on Coach redesign. See [current roadmap](phase1-security-roadmap.md).
