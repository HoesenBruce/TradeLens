# Issue #231 — deployment foundation acceptance

Validated on 2026-10-03 with Docker Desktop arm64, PostgreSQL 16 and Go 1.27.1.

## Changes

Reviewed upstream dc9bb0a and adapted its driver-layer UTC normalization.
SQLite binds times with _timezone=UTC; PostgreSQL registers a timestamp codec
on every connection, normalizing finite values before encoding. TIMESTAMPTZ
behavior and market-time display/grouping remain unchanged.

No upstream migrations were copied. TradeLens's SQLite and PostgreSQL migration
chains are retained. No new migration is required.

PostgreSQL conformance testing also exposed the existing multi-ID
DeleteTradesNotInAccount placeholder regression. Expansion now uses numbered
PostgreSQL placeholders. Existing single-ID, multi-ID and 40-trade import tests
cover the repair. Broker accounting semantics remain unchanged.

## Reproduction

From the repository root:

```sh
docker compose -f docker-compose.yml -f docker-compose.build.yml build
docker compose -f docker-compose.yml -f docker-compose.build.yml -f docker-compose.postgres.yml build
python3 scripts/deployment-smoke.py tradelens-api:local
```

The standard-library script creates uniquely named disposable containers and
volumes, then cleans them up. Requires Docker Desktop host.docker.internal
routing, Python 3.10+ and postgres:16. A temporary HTTP fixture server listens
on the host for container access. All fixtures are synthetic. HTTP bars verify
routing and authenticated contracts, not live market-data vendor behavior.

Expected output:

```text
sqlite restart+migration+persistence PASS
sqlite backup+restore+migration PASS
postgres restart+migration+persistence PASS
postgres backup+restore+migration PASS
```

The built API container is exercised over HTTP:

- First-user setup and subsequent login.
- Non-UTC execution POST, UTC same-instant deduplication, offset PATCH/read-back.
- SBI CSV import and credit-to-cash conversion: zero realized P&L, 1005 transferred basis.
- Attachment upload and byte-for-byte download after restart and restore.
- Offset news timestamp, asset links, research notes and manual prediction.
- Account/trade analytics and account value with a non-null estimate.
- HTTP provider configuration, actual authenticated upstream calls and nonempty bars.
- Fresh migrations and migrations rerun after container restart.
- Exact API read-back against seeded baselines after restart and restored startup.
- SQLite stopped-volume backup/restore, including attachments.
- PostgreSQL pg_dump/pg_restore to a new database, plus attachment-volume restore.

Also passed: go test ./..., go vet ./..., and go test ./... with
TM_TEST_DATABASE_URL pointing to a disposable PostgreSQL 16 database.
PostgreSQL tests actually ran rather than being skipped.

The Web Docker image built using pinned pnpm 11.16.0 and the frozen lockfile;
no toolchain upgrade was necessary.

## Existing data and limits

This prevents future offset writes; it does not repair historical timestamps.
Audit SQLite offset text left by previous PATCH requests before deployment:

```sql
SELECT id, executed_at FROM executions
WHERE executed_at LIKE '% -____ -____' OR executed_at LIKE '% +____ +____';
```

Other timestamp columns may need the same audit. PostgreSQL previously dropped
offsets silently; historical correction needs original-source evidence and
separate data-correction scope. Back up existing databases before updating.

Acceptance used fresh disposable databases and current migration chains, not
every possible historical/dirty database state. SQLite backup was taken with
the API stopped; live WAL-file copying is unverified. PostgreSQL logical backup
and attachments were also captured with the API stopped for consistency.

No Web UI changed. Browser interaction and mobile validation were outside this
backend deployment change. No official image publication, registry selection,
NAS/production deployment or remote CI success is implied.

## PR #239 rebase acceptance

Rebased without conflicts from base 5415d82cb629f43037f9290253e9900f0332ad7d
onto origin/main 68b6857af25d864e1537c133a0314616e3850cc0, which includes
issue #232 via merged PR #238. Rebased runtime commit: 3f3a88a.
The root .dockerignore and Compose build override are identical to this base.
Local image identities remain tradelens-api:local and tradelens-web:local.
No GHCR/publication changes were added.

Re-executed on the final source:

- Full uncached Go tests with disposable PostgreSQL 16 and go vet.
- PostgreSQL UTC timestamp test now also asserts TIMESTAMPTZ preserves the instant.
- Both SQLite and PostgreSQL Compose build commands above (API and Web).
- Deployment smoke against the final tradelens-api:local image: all four PASS lines.
- Reused #232's scratch COPY context-export method: excluded synthetic data/tmp,
  node_modules/dist/cache, .env, DB, log and key probes; retained only api, web,
  VERSION at the root. Verified 165 required resource files byte-for-byte,
  including both migration chains, calendar CSVs, OpenAPI, Web public assets,
  sources and package/lock metadata. Removed all probes afterwards.
- git diff --check.

No test cases were skipped. Five packages without test files are represented as
package-level skip events by go test -json. PostgreSQL migration, UTC storage,
multi-ID keep-list and import conformance cases actually passed.
A repeat run against the already seeded test database hit the existing news
fixture's fixed-email unique constraint; the final full run used a fresh empty
database and passed. Use a fresh disposable database for complete reruns.

The smoke cleanup now removes its anonymous PostgreSQL volume as well as its
containers. Existing unrelated containers/volumes were preserved. Historical
timestamp repair and the other limitations above still apply.
