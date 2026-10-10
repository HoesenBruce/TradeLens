# Issue #272 acceptance

Validated on 2026-10-09 using disposable fictional data. The final Web acceptance
used the production build at localhost:5372, connected to the working-tree API at
localhost:8092. Earlier state screenshots used the dev build and the same test DB.
All browser checks used the Codex built-in browser.

The implementation selectively adapts upstream TraderMemos PR #329
(`6ecb32699b68303fe71f70cd21e9c4fe82cb3fef`); upstream/main was not merged.

## Automated checks

- API: `go test ./...` and `go vet ./...` passed.
- Race checks: `go test -race ./internal/backup ./internal/jobs ./internal/api`
  passed; backup/jobs were rerun after the final identity changes.
- Web: all 189 test files / 1,081 tests passed; `pnpm run check` reported
  0 errors and 346 existing warnings; the production build passed.
- Marketing documentation build passed.
- Tests cover WAL writes during VACUUM, integrity/read-back, retention and foreign
  files, rapid unique names, persisted per-database identities, cross-process
  directory locking, permission failure, injected disk-full failure and cleanup,
  initialization failure, scheduling, owner authorization, Postgres, demo, and
  owner-to-member cache isolation.

## Real API and browser cases

| Case | Result / evidence |
| --- | --- |
| Owner manual snapshot, latest filename/count read-back | Passed, [production-current.png](production-current.png) |
| Leave/re-enter About | Applied server status retained |
| Scheduled creation and retention | Boot schedule and one-minute cadence produced snapshots; keep=3 retained newest three |
| Schedule off / manual remains available | Passed, [manual-only.png](manual-only.png) |
| Permission failure then recovery | Error card/toast and navigation dot; retry after chmod cleared error/dot, [failed.png](failed.png), [recovered.png](recovered.png) |
| Narrow Web navigation | Settings red dot visible at 390×844, [narrow-failed.png](narrow-failed.png); viewport reset afterwards |
| Pending snapshot | Button disabled during a 256 MiB snapshot, [pending.png](pending.png) |
| Competing process | Directory lock caused HTTP 409 and visible toast, [conflict.png](conflict.png) |
| Overdue snapshot then recovery | Stale state/dot changed to current after backup, [stale.png](stale.png) |
| Owner logout → member login without reload | No backup section or new admin-backup request; `/me` refetched, [production-member.png](production-member.png) |
| Ephemeral demo flag | No manual control, POST 403, no backup writes, [demo-disabled.png](demo-disabled.png) |
| Real Postgres 16 API | Unsupported guidance and no manual control, [postgres.png](postgres.png) |

Cancel/reset controls do not exist for this immediate server action. A started
snapshot intentionally survives a client disconnect. No staged settings are added.

A final large-snapshot probe observed `running=true`, rejected a second request
with 409, and completed an accounts read plus execution write in 0.225 seconds
while VACUUM was running. The published snapshot passed `PRAGMA integrity_check`.

## Container restart and restore drill

Built the current branch as a Linux arm64 API binary and ran it in a local Debian
runtime image with a persistent `/data` bind mount. Seeded one account and two
executions, took a snapshot, and confirmed status/identity survived container
restart. Stopped the container, preserved a rollback DB and an attachment-directory
sentinel separately, changed the live DB, restored the selected snapshot, removed
WAL/SHM sidecars, checked integrity, restarted and read back the original account
and both executions. Repeated snapshot/integrity checks with the final binary.

Attachments were checked using a filesystem sentinel, not an uploaded attachment.
The SQLite snapshot covers the database only; attachments and secrets need separate
backup. Keep the `.backup-id` sidecar with the database volume to retain its namespace.

## Limits

The full multi-stage API Dockerfile build was stopped during a slow base-image
download; the container drill used the locally built binary in a runtime image.
Physical disk exhaustion, power loss, Windows runtime, real NAS locking/fsync,
off-site synchronization, public Render deployment, and native iOS/Android were
not exercised. Disk-full handling uses fault injection. The demo flag is tested
locally; the demo PR was subsequently integrated from main during final review. Last-attempt/error status
is in memory and resets on restart; snapshots and the latest success remain on disk.

## Final merge review — 2026-10-10

Integrated `origin/main` at `58f6a3c1` into the existing #272 branch without rewriting
its prior commit. The sole conflict was About's moved `StatTile`; the shared
component retains main's `break-all` subline, and main's TradeLens release source,
full commit display and unavailable-release handling are preserved. Corrected the
restore path placeholder to the actual persisted backup ID. Backup core behavior,
database schema and accounting semantics are unchanged.

- `go test ./...`, `go vet ./...`, and backup/jobs race tests passed again.
- Web production build and check passed (0 errors, 347 warnings). The first
  full run timed out in the existing NewTradeDrawer batch-P&L test while builds
  ran concurrently; a second full run passed all 189 files / 1,091 tests.
- Offline deployment/release-metadata regression tests passed (10 + 3 tests).
- Built-in-browser production smoke against a fresh API passed manual backup,
  latest-file/count read-back and leave/re-entry. About links to TradeLens v0.3.1;
  [review-main-integrated.png](review-main-integrated.png) shows the live API and
  backup card. Build metadata still identifies the pre-merge HEAD because the
  integration was exercised before committing.
- **Full `api/Dockerfile` build now passed**, superseding the earlier build gap.
  Its local Linux arm64 image passed manual snapshot, integrity, user read-back,
  0600 permissions and persistent-volume restart with the same namespace/file.
- The existing stop/restore/restart drill is sufficient for this DB-only feature's
  merge gate: the backup core and schema did not change in integration. It does
  not establish uploaded-attachment recovery or a complete disaster-recovery plan.
- Review found no critical data-loss, authorization, concurrency or retention
  blocker for a single API on a local POSIX filesystem. Admin authorization is
  re-read from the DB; snapshot publication precedes retention under the directory
  lock, with separate DB namespaces. Root/host access remains trusted.
- Render demo runtime strips caller `TM_*` settings and sets jobs off. Config
  also disables jobs in demo mode, and the backup service refuses manual and
  scheduled writes. The config regression explicitly sets both jobs/backups on
  and confirms they remain ineffective in demo mode.
- Real NAS locking/atomic rename/fsync and off-site recovery remain deployment
  follow-up, not a merge blocker for local storage. Before using a network share,
  validate those guarantees or write locally and sync off-site. Do not reuse a
  copied `.backup-id` for an independent instance sharing the same backup root.

Remote checks and final mergeability are recorded on PR #285 for its final HEAD.
