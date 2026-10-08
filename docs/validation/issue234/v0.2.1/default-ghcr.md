# Default official GHCR Compose path — #234

Date: 2026-10-08. Base main: `4c4868564a64c661a23d815df95f49f05327db93`.

## Configuration

- PASS: default SQLite and PostgreSQL resolve API/Web to
  `ghcr.io/hoesenbruce/tradelens-{api,web}:latest` with an empty env file.
- PASS: source SQLite and PostgreSQL resolve to `tradelens-{api,web}:local`
  and retain build contexts/Dockerfiles from this checkout.
- PASS: custom `TM_IMAGE_REGISTRY` / `TM_IMAGE_TAG` overrides.
- PASS: service keys, ports, runtime environment, mounts, dependencies and logical
  volume keys match main. `/data/tradermemos.db`, attachments, `tm_data` and
  `tm_pg_data` remain unchanged. No database migration was edited.
- Production example pins `0.2.1`; Compose convenience default remains moving `latest`.

## Local validation

- PASS: official-image pull; isolated SQLite API health, production Web HTTP and
  same-origin `/api/v1/setup/status`. Built-in browser renders fresh setup.
- PASS: fresh source rebuild on retry (API and Web), followed by isolated startup
  with the newly built `tradelens-api:local` / `tradelens-web:local` images.
  API `/healthz`, Web `/` and same-origin `/api/v1/setup/status` all returned 200.
  Disposable containers, network and volume were removed after the checks.
  The first two builds timed out fetching npm dependencies; the next retry
  completed `pnpm install --frozen-lockfile` in 2m31s and both image builds
  exited successfully. No Dockerfile/dependency changes were needed.
  Local image IDs (not published GHCR release digests):
  API `sha256:ad809a344cd0fa0f5bccacd6df9eba482aaf972d54e8cebeb5d1e1a21ddadd95`;
  Web `sha256:e8b5a19b58d1a8937f1bc0e3149187e1253491482540b4d47c23efd2c9ff54e3`.
- PASS: Go vet, all Go tests and Go build.
- PASS: marketing production build; built-in browser inspected Deploy and followed
  its Backup & restore link, confirming rendered content. No product Web UI changed.
- PASS: `git diff --check`; release metadata regression check.
- PASS: Web check, 183 test files / 1045 tests, TypeScript and production build.
  Existing jsdom `window.scrollTo` warnings were nonfatal.
- PASS: marketing lint/typecheck and final production build.
- PASS: initial PR #251 head `ebd862915433fc456869ad63cf2a5cc045e4c2a1`
  remote API/Web/marketing and Conventional PR title checks. The suggested
  `release:` prefix was rejected; the final title uses accepted `chore:`.
  Evidence follow-up head `770073f69f3bb10d526f0289ae6f1e57868242a1` also
  passed all remote API/Web/marketing/title checks. Final evidence-update checks
  are tracked on the PR.

An initial QA configuration export retained the original Compose resource names.
Test containers briefly mounted the existing local volume before this was detected;
no business data was submitted. They were stopped/removed; the original volume
was retained (referenced by the pre-existing stopped `tradermemos-api-1` container). Subsequent QA explicitly set
independent project/network/volume names and removed only disposable resources.
This was a local QA isolation error, not a NAS operation.

NAS results are owner-provided existing evidence in [README.md](README.md), not
new tests in this change. No real financial data is recorded. Mobile was not
validated; outside current fork scope. No release/tag/image publication, workflow
enablement, package visibility change or #235 work was performed.
