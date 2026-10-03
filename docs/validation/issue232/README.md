# Issue #232 — Docker context and local image identity

Validated on 2026-10-03 with Docker Engine 29.1.2 on the native arm64 host.

## Source builds

Both commands completed successfully using the root `.dockerignore`:

```sh
docker compose -f docker-compose.yml -f docker-compose.build.yml build
docker compose -f docker-compose.yml -f docker-compose.build.yml -f docker-compose.postgres.yml build
```

Both configurations resolve API/Web images to `tradelens-api:local` and
`tradelens-web:local`. The database URLs remain SQLite and PostgreSQL respectively.
The Web build ran `tsc -b --noEmit && vp build`; the API build compiled both the
server and CLI. No Dockerfile changes or dependency changes were needed.

## Effective context

Exported the actual Docker context with a temporary Dockerfile containing:

```dockerfile
FROM scratch
COPY . /
```

```sh
docker build -f /path/to/probe/Dockerfile --output type=local,dest=/tmp/context .
```

The exported root contained only `api`, `web`, and `VERSION`. Temporary probes in
`api/data`, `api/tmp`, `web/node_modules`, `web/dist`, `web/.cache`, `web/.tanstack`,
an API `.env.*` file, a database file and a Web log were all absent. Probes were
removed after verification. Git history, docs, mobile, marketing and root local
tooling were absent too.

SQLite/PostgreSQL migrations, exchange calendar CSVs and Web public assets were
compared byte-for-byte with the source tree. Go module metadata, OpenAPI schema,
Web package/lock/workspace metadata, nginx configuration and source were retained.

## Runtime smoke

Started the built API with a disposable empty SQLite database. Startup completed
the embedded migrations and `/healthz` returned `status: ok`. Started the built
Web image with its nginx proxy targeting that API: `/` and all production assets
linked from its HTML returned HTTP 200; `/api/v1/setup/status` returned
`needs_setup: true` through the proxy.

PostgreSQL configuration and source builds were checked; a running PostgreSQL
deployment was not exercised. No Web UI behavior changed. Mobile validation is
outside this issue's scope. Remote CI is separate from these local checks.
