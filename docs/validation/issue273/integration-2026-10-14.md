# PR #292 integration acceptance — 2026-10-14

All records, identities, prices and settings are synthetic. All databases are disposable. PR #292 remains Draft; this task performs no PR merge, issue closure, release or deployment.

## Workspace and source

Original checkout: `/workspace/TradeLens`, clean `work` branch at `66e0616a4673e8602dc5830d66abd3da180431bd`. The default fetch did not include the PR branch. An initial worktree-add failed and a following merge command mistakenly fast-forwarded that clean checkout; `git reset --keep` immediately restored its original commit without discarding files. Original branch/commit and clean status were rechecked. Existing documentation and other task branches were retained.

Resumed the explicitly fetched `feat/273-review-inbox` at `a5577e7ebc864b36ae46af1ca8a155af47f0819b` in `/workspace/tm-292-integration`. Normal merge `cd2c1ede20b051dbd5250e6982eaf0c21dbdaec8` incorporates main `6016623aa3f1e088f450eafef8d8265c1dbec257`, including #305 provider sanitization and #306 Coach/server/cache isolation. Only five generated `messages.ts` catalogs conflicted; merged PO sources were recompiled. Every main catalog key/value was compared: only the existing Review alert help text differs. Review and Coach/News unavailable keys exist in all five locales.

Review Inbox business code was not rewritten. Integration additions are a production Chromium SBI review association regression and one QA-proxy correction: preserve incoming multipart Content-Type instead of replacing it with JSON. The proxy retains its synthetic FX response, and forwards the other requests to the real disposable API. The API source tree is unchanged from the merge (`f14ceb7690d17a8741765954e7f9f624ce5ebf54`). Final published head and final-head CI links are recorded in PR #292.

## SBI #307 dependency — NONBLOCKING

GitHub was fetched before work. Issue #307 had already been closed through PR #309, merged at **2026-10-14 18:36:50 Asia/Shanghai**; its merge is the fetched main above. This task did not modify its branch, accounting implementation or validation files.

Inbox membership depends on API `closed_at`, saved grade and per-account preference cutoff. It displays actual API direction and saves by trade ID. Planned direction remains a separate journal field and never selects accounting direction. Inbox does not sort executions or invoke regroup when grading.

Execution sorting/grouping can nevertheless change direction, close time, execution membership and IDs upstream: `trades/service.go` keys a trade with its first opening execution ID. Main's explicit SBI cash/long/short strategy reconciles date-only openings before closes at Tokyo date level, without claiming fill chronology. Existing `TestSBIReviewDateOnlyRowPermutations` covers all 24 orders per bucket, including equal-minute offsets and reassigned IDs; `TestSBIReviewImportPersistenceAndDedup` covers closing-first/reversed imports, stable notes and repeat regroup. Actual-time-only streams retain timestamp/ID ordering, so changing historical timestamps/identities can still change grouping and links.

Web's `Date.parse` has millisecond precision. The fixture's same-minute synthetic microseconds can become equal queue times; Inbox then orders by trade ID. An orphan close can therefore be shown before the margin round trip, without changing its semantic long direction or the record a save targets. The new browser regression locates the prepared journal by its visible review-note marker rather than assuming file/ID order proves chronology. It verifies:

- four API records, with three closed records in Backlog and the remaining short position excluded;
- planned short versus actual long;
- grade 5 saved to the intended margin trade;
- exact preservation of every other API detail field, including plans, risk/target/stop, prices, warnings and fill links;
- reverse reimport inserts zero and skips nine; the complete saved detail, grade and associations remain identical;
- reload excludes the graded record and shows Backlog (2).

**Historical regroup blocker:** before regrouping existing affected SBI accounts, take a complete backup, inventory annotations, map changed/merged IDs by execution membership, and deliberately restore and verify journals/plans/tags/attachments in a disposable copy. Current regroup prunes obsolete trade IDs and does not migrate their annotations. Stable execution IDs alone do not guarantee stable trade IDs. See [the existing #307 procedure](../issue307/README.md#historical-recalculation-and-rollback). No production or historical account was regrouped by this task.

## Commands and results

Go 1.27.0: `/workspace/.tradelens/go/bin/go`; caches: `/workspace/.tradelens/go-cache` and `/workspace/.tradelens/go-mod`. Web uses an isolated copy of the already installed locked dependency tree; no upgrades. Logs are retained under `/tmp/pr292-validation`.

| Gate | Result and evidence |
| --- | --- |
| `go test ./... -count=1` | PASS, all packages; API package 488.358s. Includes two-user/PAT/SQL narrow-patch preservation, foreign GET/PATCH rejection, forged account filtering, preference isolation, notes-only alerts, cutoff/restore and weekly deduplication. |
| `go vet ./...` | PASS. |
| Race coverage | PASS for every package and API case, using the complete inventory described below. |
| Web full coverage | All 196 files / 1,127 cases verified. Single-worker full run: 195 files / 1,126 cases pass, one unchanged DateTimePicker case exceeds 5s; that complete file then passes 8/8 with the original timeout. Earlier two-worker full run had timing failures and locale fallout; all those files pass in the single-worker run. Source-head GitHub full Web CI also passes. No assertion, case timeout or test isolation was weakened. |
| `tsc -b --noEmit` | PASS. |
| `vp check` | PASS, 0 errors / 347 existing warnings. The added Node Playwright test and proxy also pass targeted formatting/lint/type checks with 0 warnings. |
| `VITE_API=/api/v1 vp build` | PASS; production bundle is used for browser acceptance. |
| Review Inbox Chromium | PASS against production Web and the real disposable API: default/window/account filters; cancel/skip/back/re-entry/reload; failed save plus retry and one successful write; full field preservation; dismiss confirmation/cancel/read-back/restore; separate device context; loading/cold-load error/retry; keyboard; five locales at 390px; no horizontal overflow or app/console exceptions beyond intentionally injected resource failures. |
| SBI Inbox Chromium | PASS, new `review-sbi-integration.spec.ts`, real preview/commit, grade and reverse reimport as listed above. |
| Production security Chromium | FAIL / acceptance BLOCKED. Complete runs: Chromium .34 first 31/35, second 32/35; system Chromium .173 29/35. Original 30s identity-transition cases and 5s migration poll remain resource/timing failures. Isolated owner-cycle #2 rerun: default API times out, prefixed API passes. No data disclosure was demonstrated, but unfinished assertions are not accepted as a security pass. |
| `git diff --check` | PASS. |

### Race execution inventory

The exact requested `go test -race ./... -count=1` hit the API package's default 10-minute deadline at `TestCoachSettings_testConnection` (that individual case had run 1s). A full rerun with a 30-minute package budget hit the package deadline at `TestPreferencesRoundTrip` (3s). Neither run reported a data race; all other packages passed. These interrupted invocations are not claimed as successful monolithic runs.

To finish every API case without relaxing its original deadline, `go test ./internal/api -list '^Test'` enumerated all **229** top-level cases. Their unique union was distributed round-robin into eight disjoint shards, three concurrent processes. Each shard ran native `go test -race ./internal/api -count=1 -v -run '^(<its exact names>)$'`, with the **unchanged default 10-minute timeout** and all nested cases. All eight shards pass; logs verify all 229 names ran exactly once, no top-level or nested case skipped, and no data race. There are no API Example/Fuzz functions omitted by this list. No Go test or production code was altered to accelerate the suite.

[Complete source/test inventory](integration-2026-10-14/api-race-shards.json) and [all shard results](integration-2026-10-14/api-race-shard-results.json). For a rerun, read each names array, join with `|`, wrap with `^(` and `)$`, and pass as the single `-run` argument. This is aggregate complete race coverage, not a passing exit code for either interrupted monolithic command.

### Disposable production browser setup

Use the existing synthetic bars server `docs/validation/issue307/market-fixture.py` on 8093. A new temporary API database runs on 8096 with a random disposable JWT secret, temporary attachment directory, `TM_JOBS_ENABLED=false`, `TM_BACKUP_ENABLED=false`, `TM_ECON_CALENDAR_ENABLED=false`, `TM_MARKET_DATA_PROVIDER=http`, and `TM_MARKET_DATA_HTTP_BASE_URL=http://127.0.0.1:8093`. No original account database is used.

Serve the built Web through `E2E_API_URL=http://127.0.0.1:8096/api/v1 node scripts/serve-security-build.mjs` on 5189. Browser/API requests use `http://127.0.0.1:5189/api/v1`, so the eagerly mounted unrelated FX calculator receives the existing synthetic rate 1. A preliminary direct-API browser run attempted the default FX vendor; the proxy denied it and no external FX data was used. Final acceptance uses the synthetic proxy.

```sh
E2E_API_URL=http://127.0.0.1:5189/api/v1 \
E2E_BASE_URL=http://127.0.0.1:5189 \
PLAYWRIGHT_BROWSERS_PATH=/workspace/pr292-browsers \
node_modules/.bin/playwright test e2e/review-inbox.spec.ts \
  e2e/review-sbi-integration.spec.ts --timeout=120000
```

The security harness separately creates a fresh SQLite API on 8099 and local mock bars on 8100. It uses production `dist`, a real Chromium executable and `node scripts/run-security-e2e.mjs`; its existing 35 cases include three repeats of worker security and default/prefixed owner/member transitions. It retains its original timeouts and assertions.

Screenshots were visually inspected: [desktop](integration-2026-10-14/review-recent.png), [save error](integration-2026-10-14/review-error.png), [loading](integration-2026-10-14/review-loading.png), [load error](integration-2026-10-14/review-load-error.png), [English](integration-2026-10-14/review-en-390.png), [Simplified Chinese](integration-2026-10-14/review-zh-CN-390.png), [Japanese](integration-2026-10-14/review-ja-390.png), [Traditional Chinese](integration-2026-10-14/review-zh-HK-390.png), [Korean](integration-2026-10-14/review-ko-390.png), [SBI plan/direction](integration-2026-10-14/sbi-review-planned-short-actual-long.png). Earlier screenshots and acceptance reports remain intact.

The unresolved security gate is summarized in [security-gate.json](integration-2026-10-14/security-gate.json). Raw full-run and isolated-rerun logs/error snapshots remain in `/tmp/pr292-validation/security-*`. The incomplete assertions are a delivery blocker, despite the passing API isolation/race checks and source-head CI. No security test timeout, assertion or repeat count was changed.

## GitHub protection and final CI

Legacy `/branches/main/protection` returns HTTP 403 (`Resource not accessible by integration`). Separately, `/rules/branches/main` is readable and lists strict required checks **Test API**, **Test web**, **Conventional PR title**, along with PR, no-force-update/deletion and linear-main-history rules. The confirmed rule/check inventory is distinct from the legacy endpoint access limitation.

All four workflows on integration source `cd2c1ed` pass: [API](https://github.com/HoesenBruce/TradeLens/actions/runs/38552464906), [Web](https://github.com/HoesenBruce/TradeLens/actions/runs/38552464772), [title](https://github.com/HoesenBruce/TradeLens/actions/runs/38552464609), [Chromium security](https://github.com/HoesenBruce/TradeLens/actions/runs/38552464866). Final published-head CI is checked separately and reported in the updated PR description; source-head CI is not substituted for it.

Mobile apps are not validated; five-language Web at mobile width is the acceptance target. Integration result: **BLOCKED** by the incomplete local production security gate. Ready for independent review: **NO**. Next action: diagnose and pass the owner/member transitions and cache migration at their unchanged deadlines, then repeat final acceptance. Ready for merge: **NO**. Ready for Core D1: **NO**.
