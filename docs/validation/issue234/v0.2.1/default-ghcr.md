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
- PASS: source overlay startup with existing local images, same three HTTP checks.
  Fresh source rebuild and subsequent smoke: pending final validation update.
- PASS: Go vet, all Go tests and Go build.
- PASS: marketing production build; built-in browser inspected Deploy and followed
  its Backup & restore link, confirming rendered content. No product Web UI changed.
- PASS: `git diff --check`; release metadata regression check.
- Web local check/test/build and remote PR CI: pending final validation update.

An initial QA configuration export retained the original Compose resource names.
Test containers briefly mounted the existing local volume before this was detected;
no business data was submitted. They were stopped/removed; the original volume
was retained (in use by its original deployment). Subsequent QA explicitly set
independent project/network/volume names and removed only disposable resources.
This was a local QA isolation error, not a NAS operation.

NAS results are owner-provided existing evidence in [README.md](README.md), not
new tests in this change. No real financial data is recorded. Mobile was not
validated; outside current fork scope. No release/tag/image publication, workflow
enablement, package visibility change or #235 work was performed.
