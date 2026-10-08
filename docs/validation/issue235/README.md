# #235 non-publication validation — 2026-10-08

## Read-only GitHub baseline

- main: `aab42e8c9bca186066ec3a890783e6e88c126d67`
- latest stable: `v0.2.1`; annotated tag object
  `98e47f374be525f448304707b07095eb3593bd2e` resolves to
  `b78c5925778831e7d2d039a4d0139ff1ddfc6e31`.
- remote `v0.2.0`: `e638e56a184e085059020901237304ff6e0d1600`;
  remote `v0.1.0`: `6faf222ba21d8f838c0bd586c878727339375667`.
- VERSION and manifest: `0.2.1`. No open Release Please PR; open PRs were
  dependency updates #219–224. Issue #235 OPEN.
- old bootstrap object exists, but `git merge-base --is-ancestor` returns 1.
- Release Please and Docker publisher: `disabled_manually`, rechecked after edits.
- ghcr: required reviewer HoesenBruce, self-review allowed, administrator bypass
  false, custom deployment policy only tag `v*`. Automatic main caller currently
  requires owner configuration; not ready for production activation.
- Actions default workflow permissions write; PR approval setting true.
  Secret existence/content was not inspected or changed.
- Fetch found local/remote conflicts on v0.1.0 and v0.2.0. No tag was overwritten;
  baseline above comes from GitHub API, not those stale local tags.

## Results

| Check | Result | Evidence / limits |
|---|---|---|
| actionlint | PASS | actionlint installed in temporary directory; all workflows |
| YAML parsing | PASS | yaml parser; unique keys; all workflow files |
| Release Please configuration | PASS | official config schema with Ajv; uri-reference format ignored, all structural rules checked |
| Release Please engine simulation | PASS | `scripts/test_release_please.cjs`, release-please **17.6.0**, matching action v5 lockfile |
| Actual Release Please GitHub dry-run | NOT VERIFIED | No production action invoked; mocked read-only API equivalent uses real Manifest.buildPullRequests |
| Existing docs/chore history | PASS | no release candidate |
| Isolated fix / feat | PASS | 0.2.2 / 0.3.0 respectively |
| History exclusion | PASS | iterator stops at published SHA, old breaking feature never consumed |
| Changelog generation | PASS | new heading precedes retained upstream 0.13.0 text; no history deleted |
| Version extra-files | PASS | VERSION, manifest update, web and both mobile marketing files retained |
| Metadata policy | PASS | 3 Python unittest cases covering stable/prerelease, invalid metadata, manual ref, source resolution, tag drift, registry refusal |
| Source resolution | PASS (simulated API) | real local git tag + mock release API; main event resolves published tag SHA |
| Existing GHCR tags | PASS (read-only live) | both public 0.2.1 image manifests found; duplicate guard refused both |
| Duplicate/error policy | PASS (mock) | existing manifest aborts; 404 permits; 403 propagates |
| Permissions / invocation | PASS (static) | release caller grants contents read + packages write; tests consume resolved SHA; publish needs both tests and metadata |
| Event/ref behavior | PASS (source analysis) | reusable caller inherits main; environment matches GITHUB_REF; release event trigger removed to avoid PAT duplicates |
| Actual reusable call / approval / CI | NOT VERIFIED | workflows remain disabled; requires controlled owner activation |
| Real tag/Release/image creation | NOT VERIFIED | intentionally prohibited and not attempted |
| git diff --check | PASS | no whitespace errors |

Reproduce simulation without adding repository dependencies:

```sh
# Install release-please@17.6.0 in a temporary directory first.
NODE_PATH=<temporary-directory>/node_modules node scripts/test_release_please.cjs
python3 -m unittest discover -s scripts -p test_docker_release_metadata.py
actionlint
git diff --check
```

The engine warns that the existing title pattern omits optional scope/component
placeholders and that manually created v0.2.1 has no associated release PR.
Both cases are handled: the matching release SHA is still found and next-version
candidates are correct. No production dry-run success is claimed.

## Remaining acceptance

Follow the owner activation sequence in [release.md](../../release.md).
Keep both workflows disabled until reviewed. Add exactly main to ghcr branch
policies without removing reviewers or v* tags. Configure/verify the existing
RELEASE_PLEASE_TOKEN PAT or explicitly approve GITHUB_TOKEN PR checks. Preserve
required API/Web/title checks. Validate real release PR creation before merging
it. Validate tag SHA, both source-bound test jobs, approval wait, both images and
digests during the first authorized production release. Partial image publication
requires reviewed recovery; immutable tags are never silently overwritten.
