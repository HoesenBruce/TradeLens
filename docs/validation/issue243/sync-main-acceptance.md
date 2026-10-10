# PR #264 synchronization acceptance — 2026-10-09

## Source and isolation

- Old remote PR head: `b5ecb1f2960d1f52e8f58ae1d7d630a46d2fbbe8`.
- Main synchronization target: `51118ea03ca565024a7e3130cedb28faae2bdc3b`.
- Rebased application/evidence head before this documentation commit:
  `326ba9767b61d42c4648cd27befddec7b24e833d`.
- Render Live source: `3e8f7b8090673ba3d23663f2d47eecfbf590dc86`, owner-confirmed.
- Gate B PASS: owner confirmed Free, Auto-Deploy Off, no persistent disk and
  the live SOURCE matching the prior public acceptance target. Codex does not
  independently claim Dashboard configuration verification.
- Used `/private/tmp/tradelens-243`, dedicated to feat/243-render-demo. The only
  untracked file before synchronization was this task's configuration report.
  Other worktrees, including concurrent feat/272-sqlite-backups, were untouched.
- Rebase completed without conflicts. `git range-diff` matched all five original
  PR commits exactly; no demo implementation changes were added during rebase.
- Normal self-hosted Dockerfiles, Compose and GitHub workflows have no diff
  against the current main target. GHCR publishing was not changed or invoked.

## Local validation on the rebased source

- `go test ./...`: PASS; `go vet ./...`: PASS.
- `pnpm test`: PASS, 185 files / 1,057 tests.
- `python3 scripts/test_showcase.py`: PASS.
- Built deploy/demo/Dockerfile as local image tradelens-demo:243-sync: PASS.
- `DEMO_TEST_IMAGE=tradelens-demo:243-sync python3 scripts/test_demo_render.py`:
  PASS under 512 MiB and 0.5 CPU. Readiness: 2.3 s and 2.2 s on two boots;
  observed memory: 36.65 MiB / 512 MiB.
- Verified fictional initialization/reset, authentication and stale token
  rejection, 107 trades, JPY 25,000 net P&L, JPY 975,000 closing value, calendar
  filtering and empty results, news/prediction reads, and backend write denials
  including direct loopback API password-write rejection. Provider termination
  caused the supervisor to fail, as required.
- Initial Go/Web/Docker attempts hit cache permissions or a stalled base-image
  download. Authorized retries of the same checks completed successfully.

No screenshot or binary evidence was added. Existing historical screenshot
objects in the PR were not changed. No credentials, private Dashboard URLs or
deploy hooks are recorded here.

## Deployment impact relative to Render-tested source

Documentation-only changes: prior acceptance evidence and deployment guide,
release-policy documentation, repository instructions and this report.

Unrelated application changes: broker import duplicate/date resolution, OCR
wall-clock handling and test reliability fixes. Their write/OCR/import routes
remain outside the demo allowlist.

Demo-affecting application changes inherited from main: Home's market-timezone
date selection; analytics duration filtering; React-Compiler-safe money privacy
formatters across Home, Reports and trade surfaces; shared trade grouping changes
with preserved SBI strategy paths. The local container dataset and security
assertions passed, but the affected public UI was previously accepted on older
code. The demo supervisor, allowlist, demo configuration policy and fictional
fixture scripts themselves are unchanged from the Render-tested source.

**DEMO_RETEST_REQUIRED = YES.** Before merge, manually deploy the synchronized
PR source on the existing Render branch, keep Auto-Deploy Off and Free/no disk,
confirm the Live full SHA, then run affected public smoke: login/readiness,
Tokyo Home date and charts, money masking on/off, Reports/duration and trade
statistics, SBI/account value, calendar/news/predictions and backend read-only
denials. Full 15-minute idle testing is not required because startup/reset code
did not change.

## Remote gate and merge boundary

Required GitHub checks must pass on the final pushed head, never the old SHA.
The CI conclusion and exact final SHA will be recorded in a text-only PR comment
after completion. PR remains Draft until the affected public retest passes.
No automatic Render deployment, settings change, merge or issue closure is
authorized before that gate passes. Issue #243 remains open.
