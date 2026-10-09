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
locally; the separate demo PR is not part of this branch. Last-attempt/error status
is in memory and resets on restart; snapshots and the latest success remain on disk.
