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
