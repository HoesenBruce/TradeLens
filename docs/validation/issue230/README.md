# Issue #230 validation

Validated on 2026-10-03 from the current TradeLens checkout. Documentation-only
changes; application behavior, database migrations and image configuration are unchanged.
The Make help text and Compose comments now identify the default upstream images.

- `pnpm --dir marketing build`: passed; all localized MDX pages compiled.
- `bash -n`: passed for all 79 shell blocks in the 32 changed Markdown/MDX documents.
- `docker compose config --format json`: checked default, source-build, and
  source-build + PostgreSQL configurations. Default API image is upstream
  `sinhong2011/tradermemos-api`; source API image is `sinhong2011/tradelens-api`;
  PostgreSQL overlay selects the PostgreSQL URL and separate database volume.
- `git diff --check`: passed.
- Live GitHub workflow state read-back: Docker publishing and Release Please
  both remain `disabled_manually`. No publishing workflow was edited or enabled.
- Codex built-in browser against the production documentation server on port
  3230: opened all six changed deployment-related routes in each of English,
  Simplified Chinese, Traditional Chinese and Japanese (24 routes). Checked
  source-build commands and HTTPS clone examples in rendered page content.
- Exercised sidebar Deploy → Updating, then the guide's Backup & restore link.
  Destination showed the account-export limitation. Visually inspected English
  deployment, Simplified Chinese backup, Traditional Chinese updating and Japanese
  quick start. Corrected the initially visible CJK bold-markup rendering issue.

Screenshots are captures of documentation text only, with no trading data:

- `deploy-en.png`: official-image status and migration boundary.
- `backup-zh-Hans.png`: full-instance backup and account-export distinction.
- `quick-start-ja.png`: current source path and HTTPS clone.

No real instance backup/restore, Docker image build/publication, release dispatch,
NAS/cloud deployment or iOS/Android validation was performed. The shell/config
checks do not establish a successful recovery of a real database. No working API
is needed to render these static documentation pages; no application-data E2E
claim is made. Remote CI status is reported separately on the PR.
