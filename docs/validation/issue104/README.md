# Issue #104: repository rename

2026-09-29: renamed private repository ID 1365715767 to HoesenBruce/TradeLens.
The old repository API URL resolves to the same ID and new full_name. `git
ls-remote origin HEAD` returned main 04f5637; upstream remains
https://github.com/sinhong2011/TraderMemos.git and its HEAD was reachable.
The optional fork remote still identifies the separate public fork.

Updated current README badges, encoded deployment links, clone instructions,
NOTICE, contributor/deployment docs and BRAND repositoryUrl. Historical
validation records retain their original URLs, which GitHub resolves.

Validation:
- SettingsView: 21 tests passed, including repository/license/NOTICE targets.
- `pnpm check`: 0 errors, 341 existing warnings.
- `docker compose config --quiet` and `git diff --check`: passed.
- Built-in browser: working-tree Web at localhost:5188, temporary SQLite API
  at localhost:8095, signed in as issue104. Home loaded the empty QA account;
  About reported Connected. Inspected rendered About and repository, license,
  NOTICE and resource hrefs; activated repository link, left for Profile and
  re-entered About, confirming the TradeLens href remained. The private GitHub
  destination was verified separately through authenticated gh; an authenticated
  GitHub browser destination was not established. No stateful control changed,
  so inverse/cancel/reset cases do not apply.
- All 11 GitHub workflows remain active. Release chain uses relative workflow
  paths; tm-sync uses GITHUB_REPOSITORY. Docker publishing uses Docker Hub secret
  namespace and stable image names, with no GHCR references.

Remaining acceptance limitations:
- GitHub Web CI run 36554430707 did not start: annotation says recent account
  payments failed or spending limit needs increasing. CI/release success is
  unverified until billing is fixed and checks rerun.
- NAS and external cloud deployment integrations were not accessible in this
  task; live update/redeploy is unverified. Migration instructions are in
  docs/fork-deploy.md. Compose configuration passed locally; no production
  volumes, images or services were changed.
- Mobile not validated; outside current fork scope.
