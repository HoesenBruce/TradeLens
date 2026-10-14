# Issue 299: legacy AI isolation

Implementation candidate; no release, deployment, live setting change or re-enablement is authorized by this change. Core D1 review (#302) and future Coach D2 (#303) remain separate gates.

## Policy and compatibility

`coach.LegacyAvailable()` is false in this build. This is an immutable build policy, not a new owner toggle. New installations and upgraded databases both fail closed, including an owner requesting `enabled=true`. Retain existing transport settings, prompts, legacy code and rows; do not reinterpret, rewrite or delete them. Configuration still resolves DB settings over environment defaults, followed by the build policy; effective `enabled` and the public feature/capability are false. Restarts, DB edits, environment edits, stale JWT/PATs, output-version parameters and concurrent requests cannot change that policy. No new config knob or v2 implementation is introduced.

Future re-enablement requires a separate v2-capable implementation, independently approved D2 candidate and explicitly authorized configuration. Removing one route gate is insufficient: the direct generators, internal transport/fallback and persistence also reject legacy execution.

### Entry-point inventory

| Entry | Disabled behavior / boundary |
| --- | --- |
| `GET /api/v1/capabilities/coach` | Authenticated JWT/PAT; fixed `available=false`, `news_analysis_available=false`, `reason=unavailable`; no settings, model or prompt; `no-store`. |
| `POST /api/v1/trades/:id/coach` | Authenticate, owned trade lookup, fixed compatibility JSON `source=off`, empty notes, server unavailable message. No context/provider/save. |
| `POST /api/v1/trades/:id/coach/stream` | Same check before opening SSE; returns fixed JSON understood by old stream client. No note/progress/model bytes. Version parameters cannot bypass it. |
| `GET /api/v1/trades/:id/coach/reviews` | Owned trade and user-scoped rows; only validated UUID/time, empty notes, `status=legacy_withheld`, fixed message. Never decode/emit legacy notes, action or model; malformed content also withheld. Invalid ID/time rows skipped without raw diagnostics. `no-store`. |
| `GET/PUT /api/v1/settings/coach` | Existing owner-only transport configuration retained; effective enabled is always false. Saving inactive values does not grant generation. This private settings response is distinct from capability. |
| `POST /api/v1/settings/coach/test`, `/models` | Owner-only; fixed `503 ai_unavailable` before any upstream connection/model metadata. |
| `coach.GenerateReview`, `GenerateReviewStream`, `AnalyzeNews` | Direct service calls reject before constructing input/upstream requests. |
| `requestContent`, `requestReview`, `streamReview`, fallback retries | Transport guard rejects even package-local alternate calls, regardless of `VisionConfig.Ready`. No fallback can start. |
| `saveCoachReview` | Rejects before marshaling or storage, including direct handler-package calls. |
| Jobs | Existing jobs have no Coach generator/history callers. No new job or alternate AI entry point added. |
| Trade exports, aggregate public shares, news exports | No `coach_reviews` reader or Coach DTO path; old Coach samples cannot enter these serializers. Trusted operator database backups still preserve raw rows and remain sensitive. |
| Web trade detail/settings | Uses non-sensitive capability, never owner settings to decide availability. This build does not fetch/interpret legacy Coach history or mutation/query-cache results. Rule-based notes and manual journal remain usable. |
| Web cache / worker | AI API paths use `NetworkOnly`; activation deletes matching entries across existing caches. Security worker automatically activates and claims open clients instead of waiting for an update button. |

News is a separate inventory: `POST /news/:id/analyze` uses shared unsafe free-text provider output and is temporarily unavailable. `POST /news/:id/analysis/accept` is an authenticated owned, explicitly submitted manual review and retains conflict/validation/transaction checks; News CRUD, manual assets/predictions, exports and validation are independent. Historical News records are normal owned user records; this change neither certifies old AI news prose nor silently rewrites it. Its future AI output boundary must be accepted independently, without requiring Coach C/D2. Existing manual records remain available.

## Validation

Only synthetic data and disposable SQLite were used.

- Real-route owner/member JWT and PAT: JSON, SSE-compatible off response, v2 parameter attempts, history, foreign-trade/news denial, anonymous denial, owner connection/model endpoint denial. Spy provider receives zero calls. Database digest comparisons cover review rows and unrelated tables; only documented PAT usage bookkeeping is normalized out.
- Persisted DB enabled/disabled overlays with env-equivalent enabled defaults and reconstructed servers retain old tokens safely; actual `config.Load()` with `TM_COACH_ENABLED` true/false and no DB settings row also fails closed. Eight concurrent stream requests emit no provider event and write nothing. Direct service, internal fallback/transport and persistence calls are denied.
- Legacy valid and malformed rows include arbitrary synthetic private display/model/action markers and remain unchanged. Preservation checks compare all row-column digests, not counts alone.
- New Web hook ignores seeded legacy query data and enabled private settings across logout and a new sign-in; it only calls capability. No old model text is interpreted.
- Built-in browser: real API, enabled env and seeded old Coach row; fixed unavailability visible, old marker absent, rules shown; manual journal save/read-back/re-entry passed. News manual surface and unavailable state visually inspected. Screenshots in this folder.
- Chromium regression: real API unavailable trade/news/settings; Coach collapse/expand; manual News cancel/re-entry/save/reload. Production-worker test reproduces old cached raw-history replay, updates without a `SKIP_WAITING` message or tab close, checks cache purge and safe network read, then verifies offline history is blocked. Its old worker is a deliberately minimal cache-replay fixture, not a claim to exhaust every deployed historical worker version.
- Full Go tests, vet and race; Web tests, check/typecheck/build. Exact-head CI recorded in the PR, not inferred from an older head.

The immutable gate admits no in-flight generation in this binary, so toggling config requires no speculative dynamic cancellation machinery. Previously running **older binaries** cannot be stopped by a source patch: deployment must terminate/drain old AI processes/connections before exposing the safe replacement. No production process was touched.

## Safe rollout / rollback

With separate operational authorization, stop old unsafe AI processes/streams, deploy this or another containment-capable backend first, and then the compatible Web/worker. Verify backend policy and worker activation before declaring client cache cutover complete. A client disconnected from updates can retain its old worker until it reconnects. Network isolation/process shutdown is required if an unsafe binary remains reachable. Do not roll back to a binary that allows legacy generation/history. No destructive schema migration or automatic backup restore is needed here; retain existing DB rows and inactive config.

Previously copied, displayed, downloaded or externally stored bytes cannot be recalled. Worker activation clears retained AI response caches and blocks future replay, but it does not erase an old tab's already rendered DOM/JS memory; reload that tab for the new UI. Old disconnected clients and third-party caches outside this origin require their own operational cutover. These are delivery limits, not a confidentiality guarantee for past disclosure.

Mobile and PostgreSQL runtime were not exercised; mobile is outside current fork scope. No SQL/schema change is made, and the common service/API guard applies to both store drivers. Review Inbox (#292) is not present on this baseline; no invented Inbox implementation or acceptance claim is included.

### Re-running browser regressions

Use a disposable API and an authenticated synthetic user. `E2E_COACH_FIXTURE` is a local JSON file with `trade` and `news` UUIDs: a closed synthetic trade with a raw legacy review row and the journal text `Issue 299 manual review remains usable`, plus an editable synthetic News record. Use the built-in browser to write/read that journal first. Set `E2E_EMAIL`, `E2E_PASSWORD`, `E2E_API_URL`, `E2E_WEB_URL`, and `E2E_BASE_URL` for this environment.

Build Web and freeze the complete `dist` directory to `E2E_WORKER_DIST` before the upgrade test when a dev server is also running: the Serwist development middleware serves then removes its generated worker file from `dist`. Run `playwright test e2e/coach-isolation.spec.ts e2e/news-analysis.spec.ts`. The worker-only case needs no authentication or live API; it serves the real production artifact locally and deliberately seeds a minimal old cache worker. The actual deployed-worker variants and production proxy are separately verified during rollout.

## Main integration and provider cache boundary (2026-10-14)

PR #305 was independently accepted and squash-merged as `021ffa091ca9a0e4c29a3d5acdffeb12d54a2f01`. This branch integrates that main with a normal merge; no history rewrite. Its provider configuration matrix now expects disabled Coach test/models to return 503 with zero provider calls, while OCR retains the diagnostic tests.

Independent integration testing found that the production security worker cleared Coach caches but retained old OCR owner settings and Flex/notification error readbacks. A synthetic cached OCR response was actually returned offline after worker upgrade. The fix reuses the worker's NetworkOnly and activation purge for `/settings/ocr`, `/flex-sync`, `/accounts/:id/flex-sync`, and `/settings/alert-channels` as well as the existing Coach/News AI paths. These settings/status surfaces now require network access; other Core offline behavior is unchanged. The permanent Chromium worker regression seeds all five representative paths, verifies automatic activation removes them, and verifies offline fetches fail.

News AI generation remains temporarily disabled: the legacy shared transport and free-text success fields have no independently accepted confidentiality boundary. Manual News CRUD, assets, predictions/history, review acceptance and exports remain available with existing ownership/validation. Historical News user records are preserved; their prose is not newly certified or sanitized by this containment change. No narrow provider-backed substitute is introduced without a separate output-boundary acceptance.

Validation of the integrated head and exact-head CI is recorded in PR #306. This security integration fix keeps the PR Draft for a separate final independent review. No merge of #306, release, deployment, issue closure, tag, image publication or production database change.
