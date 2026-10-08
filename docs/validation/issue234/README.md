# #234 controlled first-release checklist

> Historical pre-publication plan. Completed v0.2.1 registry/runtime and owner-provided
> NAS results are in [v0.2.1/README.md](v0.2.1/README.md); the final default-image
> switch is recorded in [v0.2.1/default-ghcr.md](v0.2.1/default-ghcr.md).
> PENDING entries below describe the original plan, not current release availability.

Preparation only. No registry/runtime result is established by this document.
Keep #233 and #234 open until their respective evidence requirements pass.

## Readiness snapshot (2026-10-03)

READY:
- #240 squash merged: c9bfd28345a0f2fd702674c079d17f3f7c6a3b14.
- API, Web and both PR-title checks completed successfully before merge.
- GHCR-only publisher, shared version/full SHA/build time, paired digest artifacts.
- Policy regression and actionlint passed; no UI changed.
- Existing `scripts/deployment-smoke.py` accepts a registry API digest and tests
  disposable SQLite/PostgreSQL, synthetic HTTP provider, features, restart/restore.

OWNER ACTION REQUIRED:
- Environment API returned zero environments; GET environments/ghcr returned 404.
  Create/confirm `ghcr`, required reviewer, prevent self-review where supported,
  and deployment tag policy permitting only the approved release tag. Verify
  account/plan support for protection rules; merely naming an environment is not a gate.
- Choose an approved release version and source SHA. VERSION is 0.1.0; only
  v0.1.0 exists and points to f17f56db2dd42a5e95330953a579c12922804d88,
  which predates #240. No GitHub Releases exist. Do not use that old tag for this
  publisher or retag it. Recommend a new version (e.g. 0.1.1), with owner-approved
  version-file updates and exact reviewed commit, followed by a new tag/Release.
- Confirm Actions package-write policy and eventual package repository linkage.
- Clear existing publication/privacy/provenance gates separately; green CI does
  not clear those owner decisions.

PENDING FIRST PUBLICATION:
- Both packages, visibility Public, anonymous pulls, real tags/OCI/manifests,
  runtime acceptance on both architectures and databases, NAS update acceptance.
- Docker and Release Please are disabled_manually. This preparation must not
  enable either, create releases/tags, change visibility or publish images.

## Execution order (future authorized publication session)

1. Owner clears gates above; approve exact version, release tag and 40-character
   source SHA. Ensure VERSION and product version metadata agree. Check CI at
   that SHA, not an arbitrary later main HEAD.
2. While publishers remain disabled, create the approved tag and published
   non-draft GitHub Release. For stable: prerelease=false and latest stable release
   must match. Review all other release-triggered workflows remain disabled
   (Release Please, tm-sync and mobile); do not activate them.
3. After explicit authorization, enable only Docker publishing for the controlled
   run. Manually dispatch `docker-publish.yml` with `--ref v<version>` and input
   `version=<version>`. Do not dispatch from arbitrary main HEAD. Confirm resolved
   SHA equals approved tag commit, tests pass, then reviewer approves `ghcr`.
   Revalidation after approval must pass. Keep other release creation frozen
   through publication: GitHub Release state is external to workflow concurrency.
4. Retain run URL, both digest artifacts and summary outside their 90-day retention.
   A partial matrix success is incomplete; never switch Compose defaults then.
5. Owner explicitly sets each package Public in GitHub package settings and
   records visibility, repository link and Actions write access. Workflow does
   not automate this. Verify both packages, not repository visibility alone.
6. Fill the variables below from the recorded artifacts and run anonymous pulls
   and manifest inspection with a newly empty Docker config.

```sh
API_DIGEST='sha256:<replace-with-recorded-api-digest>'
WEB_DIGEST='sha256:<replace-with-recorded-web-digest>'
qa_config=$(mktemp -d)
DOCKER_CONFIG="$qa_config" docker pull "ghcr.io/hoesenbruce/tradelens-api@$API_DIGEST"
DOCKER_CONFIG="$qa_config" docker pull "ghcr.io/hoesenbruce/tradelens-web@$WEB_DIGEST"
DOCKER_CONFIG="$qa_config" docker buildx imagetools inspect "ghcr.io/hoesenbruce/tradelens-api@$API_DIGEST"
DOCKER_CONFIG="$qa_config" docker buildx imagetools inspect --raw "ghcr.io/hoesenbruce/tradelens-api@$API_DIGEST"
DOCKER_CONFIG="$qa_config" docker buildx imagetools inspect "ghcr.io/hoesenbruce/tradelens-web@$WEB_DIGEST"
DOCKER_CONFIG="$qa_config" docker buildx imagetools inspect --raw "ghcr.io/hoesenbruce/tradelens-web@$WEB_DIGEST"
```

7. Confirm actual registry manifests include linux/amd64 AND linux/arm64 for
   both images. Record per-platform child digests; attestation descriptors with
   unknown platform are not runtime architectures. Inspect pulled per-platform
   image labels with `docker image inspect`; source must be TradeLens, revision
   full approved SHA, version exact approved version. Verify all expected tag
   references resolve to the recorded paired index digests.
8. Run API foundation checks against registry digest, never local build image:

```sh
python3 scripts/deployment-smoke.py "ghcr.io/hoesenbruce/tradelens-api@$API_DIGEST"
```

   Save all PASS output. The script uses disposable data and synthetic provider,
   stops API before backup, restores PostgreSQL into a new database, restores
   attachment files and checks startup. It is API evidence, not Web E2E or a
   live-provider result. It uses the host's default Docker platform.
9. Run Web/API together from both recorded digests in a temporary Compose
   override, separate project and fresh volumes. Set pull_policy=always; use no
   build overlay and no production .env/secrets. Inspect `docker compose config`
   before startup to ensure both services resolve to exact GHCR digests. Use
   alternate loopback ports if normal ports are occupied. For PostgreSQL include
   docker-compose.postgres.yml, and keep its database isolated. Repeat browser
   acceptance below for each database mode.
10. Record native host and Docker engine architecture. Current preparation host
    reports arm64; verify again at execution. Run both images on native platform,
    then on a real second-architecture host where practical. If using QEMU,
    explicitly set linux/amd64 or linux/arm64 in the disposable Compose override
    and record engine/emulator versions and startup/feature results. Explicitly
    mark any unexecuted architecture incomplete. Manifest presence is not runtime
    evidence. Build-time QEMU is not runtime acceptance.
11. After all registry/runtime evidence passes, prepare a separate reviewed change
    switching prebuilt Compose defaults and deployment docs to GHCR. Retain source
    build fallbacks. Validate NAS/external update with a rehearsed isolated copy
    and verified complete backup; production changes need explicit authorization.
    Preserve existing volume identities; never alternate upstream/fork against
    the same database. Feed actual NAS evidence into #98.
12. Disable Docker publishing again after the controlled run unless an explicit
    reviewed ongoing publication policy says otherwise. Release Please remains
    disabled. Evaluate #233 and #234 separately against evidence below.

## Browser/runtime acceptance matrix

All data is synthetic. Run every row for SQLite and PostgreSQL using published
API + Web digests. Use built-in Browser and retain rendered screenshots where
useful. Record expected values before asserting results.

| Case | Required evidence in each database mode |
|---|---|
| Fresh deploy | health, setup wizard, owner creation, login/logout/login; PostgreSQL fresh migrations |
| Import | synthetic broker file, imported execution count, read-back and expected grouped trades |
| SBI / 現引 | source-defined synthetic conversion, custody transfer/cost basis and expected realized P&L |
| Analytics/value | known account totals, currency and valuation baseline; rendered read-back |
| Attachments | upload/open/download, exact bytes/hash, restart and restored download |
| News/research | create/read/edit synthetic thesis/prediction/research, re-entry and persistence |
| HTTP market data | synthetic HTTP provider configuration, real request log, chart/value read-back; no invented live-vendor claim |
| Restart | restart published containers; same owner/trades/news/attachments/analytics |
| Complete backup | stop API for consistency; SQLite DB+attachments or pg_dump+attachments; retain temporary deployment configuration |
| Restore | fresh isolated volumes/database, pg_restore for PostgreSQL, attachments restored, successful migrations/startup/login and full read-back |

Exercise applicable inverse, cancel, reset, re-entry and both-direction transitions
for controls used. Do not treat old #231 local-image evidence as this release's
runtime evidence. See [#231 evidence](../issue231/README.md) and
[backup/restore guide](../../../marketing/content/docs/self-hosting/backup-restore.mdx).

## Evidence record (fill after execution)

| Field | Result |
|---|---|
| Approved version / tag / full SHA | PENDING |
| GitHub Release and workflow run URLs | PENDING |
| ghcr protection / deployment policy read-back | OWNER ACTION REQUIRED |
| API index digest / per-platform digests | PENDING |
| Web index digest / per-platform digests | PENDING |
| Actual manifest architectures / OCI labels | PENDING |
| API + Web Public visibility evidence | OWNER ACTION REQUIRED after publication |
| Anonymous API + Web digest pull logs | PENDING |
| Stable / prerelease / manual actual tag evidence | PENDING; unit tests are not registry evidence |
| SQLite / PostgreSQL API and browser evidence | PENDING |
| Native / second architecture runtime method and results | PENDING |
| Restart / complete backup / restored startup evidence | PENDING |
| NAS/external update / #98 reference | PENDING |
| Workflow disabled-state read-back after run | PENDING |

#233 closure needs BOTH packages created/public, anonymous digest pulls, real
stable/prerelease/manual policy evidence, paired digests and source/version alignment.
Do not create extra stable releases merely for a test; separately authorize the
policy verification runs, reject old stable backfills, and verify prerelease/manual
runs leave stable aliases unchanged. #234 additionally needs actual registry,
all runtime/persistence/restore cases, validated prebuilt-default switch and NAS
procedure. Neither issue closes from CI/actionlint alone.
