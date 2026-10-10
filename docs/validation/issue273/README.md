# Issue #273: Review inbox

Selective adaptation of TraderMemos #342/#344; no upstream branch merge or schema migration.

- `/review` is available from navigation. Recent defaults to a rolling 14-day close-time window (configurable 1–90 days); older closed trades appear in Backlog. Queues use the global account selection, including multiple accounts, but deliberately ignore global date/symbol/status/tag filters. Oldest closes are reviewed first.
- Only execution grades 1–5 count as reviewed. Notes alone do not. Save uses a narrow journal PATCH (execution grade, review notes, mistake tags). TradeLens's journal parser/builder preserve planned direction separately from execution direction, entry/exit sections, session and legacy content. Unchanged review notes keep the original notes bytes. Other tags, setups, risk and target/stop fields remain untouched.
- Skip and Back reload persisted values, dropping uncommitted edits. Cancel edits restores the current trade. Save advances only on success; an error retains the draft for retry. Reopening the page starts from currently ungraded trades.
- Dismiss backlog requires confirmation and records `reviewBacklogCutoff:<account_id>` in the authenticated user's merged preferences for **each selected account**. It excludes closes at/before the current recent-window boundary across devices, without grading/deleting trades. Restore writes null for the selected accounts only. A later wider window still respects that saved cutoff. Global selection means all current non-backtest accounts, not future accounts. Accounts not selected are unaffected.
- The alert job uses the same grade and per-account cutoff rules. Alerts keep their own configured age threshold and user-wide scope; their number need not equal the current account/window queue. Previously delivered weekly alerts are not rewritten. Notes-only trades may now produce future alerts.
- English, Simplified/Traditional Chinese, Japanese and Korean labels are included. No accounting or historical statistics recalculation.

## Validation

- Go full suite and `go vet ./...` passed; regression tests cover notes-only grading, cutoff parsing, account-specific exclusions and restoring preferences without changing another account/user.
- Web suite: 1,068 tests passed initially; two catalog-loading tests encountered temporary generated-file removal during concurrent compilation. All affected catalog tests were rerun successfully after generation completed, along with inbox/journal tests and a new five-locale regression. Typecheck and Vite+ check have no errors; existing lint warnings remain.
- Playwright ran against a throwaway SQLite API from this branch, with unused external market/calendar providers disabled and a fresh browser context. Verified recent/backlog/window reset, grade gate, notes/mistake prefill, non-mistake exclusion, cancel, skip/back/end/re-entry, failed PATCH followed by retry, save-and-next, API field-level before/after equality, dismissal confirmation/cancel/read-back/restore, multi-account scope, Chinese/Japanese rendering. Browser page-error list was empty.
- Screenshots: [recent queue](review-recent.png), [retained draft after error](review-error.png), [Chinese multi-account view](review-zh.png).

Mobile not validated; outside current fork scope. Weekly-focus features and upstream keyboard shortcuts are outside this issue's acceptance scope. The existing weekly-alert deduplication policy is retained.

## Cloud integration acceptance — 2026-10-10

Initial PR head: `ba8563e863bb8318841a2231ef30c6e2501d1502`. Initial and final fetched main: `005dd367879b1564e2add665a73dee3e0c26a333`. Accepted source/test commit: `2ea8e09ff3cc5730500791c563442e960cef9ea7`; the following documentation commit only retains this report and screenshots. The final PR head and its remote CI results are recorded in PR #292.

The initial `/workspace/TradeLens` checkout was clean on `work` at `58f6a3c1a80cc195c3650fd48ef71680684ccb50`. Its only worktree was left on that branch and stayed clean. A new `/workspace/tm-273` worktree uses the existing `feat/273-review-inbox` branch. Other local/remote branches, including Cloud development and documentation branches, were inspected and left untouched. No local #272/#244 worktree was attached. No reset, clean, deletion, force push, upstream merge, deployment, release, PR merge or Ready-for-Review transition was performed.

A normal merge (`f96b59e`) integrates origin/main, preserving #272 backups. Conflicts were exclusively the appended translation entries in `messages.po` and generated `messages.ts` for en, zh-CN, zh-HK, ja and ko. Both sets of entries were retained and Lingui recompiled. No business-code conflicts remained. Issue #273 and upstream TraderMemos #342/#344 were reviewed; fork-specific account scoping and journal preservation remain intact.

### Commands and results

Commands run in `api/` use Go 1.27.0 installed under `/tmp/go`; `GOCACHE=/tmp/273-gocache GOMODCACHE=/tmp/273-gomod` isolate caches. Web commands run in `web/` using the already installed locked dependency tree; no dependency upgrades were made. Direct executables avoid pnpm attempting a runtime/store installation outside the writable workspace.

- `/tmp/go/bin/go test ./...` — PASS, all packages, including SBI cash/margin, 現引/現渡, account-value, importer, positions, trade, alert and backup regressions.
- `/tmp/go/bin/go vet ./...` — PASS.
- `/tmp/go/bin/go test ./internal/api -run TestReviewAcceptanceIsolationAndAlertPersistence -v` — PASS. This test is also included in the full suite.
- `./node_modules/.bin/vp test run` — PASS, 190 files / 1,095 tests. Existing jsdom `window.scrollTo` diagnostics do not fail tests.
- `./node_modules/.bin/tsc -b --noEmit` — PASS.
- `./node_modules/.bin/vp check` — PASS, 0 errors; existing 347 lint warnings remain.
- `./node_modules/.bin/vp build` — PASS. TypeScript was checked separately, matching the build script's two steps.
- `E2E_API_URL=http://localhost:8095/api/v1 E2E_BASE_URL=http://localhost:5173 PLAYWRIGHT_BROWSERS_PATH=/workspace/pw-browsers ./node_modules/.bin/playwright test e2e/review-inbox.spec.ts --timeout=120000` — PASS, Chromium.
- `git diff --check` — PASS.

The final browser run uses a newly created disposable SQLite database `/tmp/273-qa-final-head.db`. API startup:

```sh
TM_HTTP_PORT=8095 TM_DB_PATH=/tmp/273-qa-final-head.db \
TM_ATTACH_DIR=/tmp/273-attachments-final-head \
TM_JWT_SECRET=$(openssl rand -hex 32) TM_ALLOW_REGISTRATION=true \
TM_BACKUP_ENABLED=false TM_MARKET_DATA_ENABLED=false \
TM_ECON_CALENDAR_ENABLED=false TM_CORS_ORIGINS=http://localhost:5173 \
GOCACHE=/tmp/273-gocache GOMODCACHE=/tmp/273-gomod \
/tmp/go/bin/go run ./cmd/server
```

Web startup: `VITE_E2E=1 ./node_modules/.bin/vp dev --host 127.0.0.1`. The built-in browser was unavailable; Playwright Chromium was downloaded without installing system dependencies. Early fixture attempts failed because market-data routing was enabled or explicit local CORS origins were missing; the corrected disposable configuration above passed. No security controls were changed.

### Acceptance evidence

- Default 14-day closed/ungraded queue, notes-only eligibility, configurable 30-day window/reset, single/multiple account selection, skipped queue completion, cancel, back, save-and-next, reload and navigation re-entry passed. Exact date-boundary/open/future/invalid-grade cases are additionally pinned by `reviewInbox.test.ts`.
- Failed PATCH retains edits, successful retry advances, and write count is exactly one failed request plus one successful request. Full API detail before/after equality excludes only notes/tags/grade; notes equal the original with only the review text replaced. Planned short versus actual long, legacy/session/entry/exit text, two setups, risk, target/stop and non-mistake tags are preserved.
- Dismiss cancel/confirm, account-specific cutoff read-back, ungraded backlog preservation, refresh, restoration and a separate browser context passed. Loading and cold-load error/retry passed in a fresh browser without service-worker support, preventing existing offline caches from masking the injected error. Ordinary browser workflows retain service-worker support. Offline cached reads are inherited behavior, not a new notification or caching architecture.
- `review_acceptance_test.go` uses two users/accounts and a fresh database. Foreign trade GET/PATCH return 404; unauthenticated calls return 401; forged single/mixed account filters never expose the other trade; an ordinary member receives 403 for owner-only backup access. Direct SQL confirms no unauthorized journal write. Preferences with forged account keys stay in the caller's user row, cannot suppress another user's alert, and do not modify the other user's trade.
- The alert service evaluates notes-only trades, stored per-account cutoff, restore and another user; stored events remain identical after restoration because weekly deduplication is preserved. Alert/inbox age windows intentionally differ as documented above.
- Desktop and all five supported locales at 390px passed without horizontal overflow; keyboard Enter activates the focused grade and cancel restores it. Browser page-error arrays are empty. Screenshots were visually inspected.
- No new dependency versions, migrations, binaries or secrets are included. Newly merged backup files are from origin/main, not unrelated new feature work. Mobile and the broader #244 security audit remain outside this acceptance.

Current Cloud screenshots: [desktop](cloud-review-recent.png), [save error](cloud-review-error.png), [loading](cloud-review-loading.png), [load error](cloud-review-load-error.png), [English 390px](cloud-review-en-390.png), [Simplified Chinese 390px](cloud-review-zh-CN-390.png), [Japanese 390px](cloud-review-ja-390.png), [Traditional Chinese 390px](cloud-review-zh-HK-390.png), [Korean 390px](cloud-review-ko-390.png).
