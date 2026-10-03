# Issue #233 — GHCR distribution implementation

- Publication targets only ghcr.io/hoesenbruce/tradelens-api and tradelens-web.
- Python policy regression: stable, prerelease, manual, mismatched SHA, old
  stable, draft, incorrect prerelease flag, missing release and malformed version.
- actionlint passed for docker-publish.yml and release-please.yml.
- git diff --check passed. No application UI changed.
- Live workflow read-back on 2026-10-03: Docker publishing and Release Please
  remain disabled_manually. No workflow enabled or image published.
- Package visibility must explicitly become public for both packages after the
  first controlled publication. Anonymous digest pulls, multiarch manifests and
  paired image digest evidence remain pending. See docs/release.md.
- No mobile validation: outside current fork scope.

This is implementation evidence, not first-release operational acceptance.
