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
docker build -f api/Dockerfile -t tradelens-231-api:qa .
docker build -f web/Dockerfile -t tradelens-231-web:qa .
python3 scripts/deployment-smoke.py tradelens-231-api:qa
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
