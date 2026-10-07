# Controlled publication evidence: v0.2.1

Date: 2026-10-07. Source SHA: b78c5925778831e7d2d039a4d0139ff1ddfc6e31.
Release: https://github.com/HoesenBruce/TradeLens/releases/tag/v0.2.1
Run: https://github.com/HoesenBruce/TradeLens/actions/runs/37590176429

All metadata, API/Web CI and both publication jobs succeeded after owner approval.
The v0.2.0 tag remains unchanged; its earlier publication failed before registry
login/build/push. #247 fixed the missing tag refs in the post-approval checkout;
#248 synchronized source metadata to 0.2.1.

## Paired image index digests

- ghcr.io/hoesenbruce/tradelens-api@sha256:9bf8d718c6c6df96fc02433e82bcb5584dff836d020f800b38a12e01a225263e
- ghcr.io/hoesenbruce/tradelens-web@sha256:a3694f5fc79aac7291948d829cda845e8e286fe2ce9e2fe7eeed9fe946d35f72

Both digest artifacts record source SHA above and version 0.2.1.

## Verified

- Both images pulled successfully by digest using a newly empty Docker config,
  with no login or credential helper. This verifies anonymous registry access.
- Actual anonymous registry manifests are saved in api-manifest.json and
  web-manifest.json. Both include linux/amd64 and linux/arm64. unknown/unknown
  descriptors are attestations and do not count as runtime architectures.
- Native Docker engine: aarch64. Pulled arm64 image labels for BOTH images:
  source=https://github.com/HoesenBruce/TradeLens;
  revision=b78c5925778831e7d2d039a4d0139ff1ddfc6e31; version=0.2.1.
- Existing scripts/deployment-smoke.py ran against the published API index digest,
  not a local build. Disposable SQLite/PostgreSQL with synthetic data/provider:
  all four restart/persistence and backup/restore PASS lines in api-smoke.log.
  Includes stopped API backup, PostgreSQL pg_dump/restore, attachment restoration
  and restored startup checks. This is API evidence, not browser E2E/live vendor evidence.
- Docker publishing and Release Please remain disabled_manually.

## Pending acceptance

- Package settings visibility field: API access returns 403 (missing read:packages).
  Anonymous access is verified; explicit owner Public visibility read-back pending.
- Actual stable aliases/full-SHA tag digest mapping verification, and real
  prerelease/manual-no-version policy evidence.
- amd64 image labels/runtime; actual Web/API browser acceptance for both databases.
- Complete #234 synthetic import/SBI/account/news/HTTP-provider rendered acceptance.
- NAS/external update procedure and separately reviewed Compose default switch.

#233/#234 remain open. #235 is not started. No visibility setting, Compose default,
production database, mobile or tm-sync release was changed by this validation.
