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

## Final release-path review — 2026-10-08

Refreshed PR #252: OPEN/Draft/MERGEABLE at `1b1270f765fea15db3f71033120560fce915a8b9`,
base main unchanged. All API/Web/title CI checks passed on that head. Latest
stable and tag SHAs remain as above; both publication workflows remain disabled.
Review changes are limited to release workflows, regression checks and docs.

### Confirmed gap fixed

Initial metadata previously compared tag/version files but had no independently
supplied Release Please SHA. Release Please now exports its created release `sha`
and passes required `source_sha` to Docker. Initial validation compares tag commit
to that value (manual dispatch compares to the tag-context `github.sha`). Post-
approval validation compares again to the metadata commit. Missing automatic
source SHA cannot enter the metadata job. The action v5 source outputs every
CreatedRelease property including sha for the root package.

Caller context is main ref / push event / caller commit SHA. Inputs.version is
the action's created-release version. Metadata.commit is the explicitly resolved
published tag commit after independent SHA, VERSION and manifest comparisons.
API/Web checkout, Docker checkout, build args and OCI revision all bind to this
metadata commit. Reusable workflows cannot elevate caller token permissions.
Publisher receives packages write; metadata/tests receive contents read.

### Owner gates corrected

The previous docs overstated main branch protection. Live `rulesets` returns `[]`;
`branches/main/protection` returns 404 "Branch not protected". Required checks
are therefore **not currently enforced** despite this PR's successful CI.
Owner must configure and verify main protection before release enablement.
Current ghcr reviewer remains HoesenBruce, administrator bypass false, self-review
allowed; only v* tag policy exists. Option A (exact main plus v* and reviewer) is
selected; tag-context Option B needs different event/dispatch authentication and
adds delivery/duplicate risks. Exact Settings steps are in release.md.

GITHUB_TOKEN PR-event approval behavior is supported by current GitHub docs, but
has not been exercised with Release Please on this repository. Human-created PR
CI is not evidence for bot-created PR CI. PAT remains the simplest existing input;
App installation tokens are an alternative requiring a separate workflow change.
No secret was read, printed or changed.

### Rerun checks

| Check | Result |
|---|---|
| actionlint; workflow YAML; official config schema | PASS |
| Source SHA wiring, approval dependency and trigger graph assertions | PASS (STATIC) |
| Real engine 17.6.0, mocked API: docs/chore none, fix 0.2.2, feat 0.3.0 | PASS (SIMULATED) |
| Breaking `feat!` from 0.2.1 → 1.0.0 | PASS (SIMULATED); bump-minor-pre-major is not enabled |
| Version/changelog and exclusion of old breaking history | PASS (SIMULATED) |
| Metadata/tag ref/SHA mismatch tests | PASS (SIMULATED API, real local git) |
| Exact-version and full-SHA duplicate probes; token 404; manifest 401/403/429/500 | PASS (SIMULATED) |
| git diff --check | PASS |
| Actual main reusable invocation, token-generated PR checks, deployments and releases | NOT VERIFIED; prohibited in this review |

### Residual risks and recovery

The registry probe is not an atomic lock. Repository concurrency cannot prevent
an external package writer creating/overwriting a tag between inspection/build/
push. Nor does it prevent a new external stable Release after validation.
Official packages are public; private-package live behavior is NOT VERIFIED.
Owner must restrict publication writers and serialize out-of-band release work,
or require stronger registry/promotion enforcement before activation.

Both API-only and Web-only success, ambiguous push failures, approval expiry,
new stable while waiting, cancellation, same-release reruns and an already
published SHA are covered in release.md's recovery table. Never infer registry
absence from workflow failure. Read both digests first. Partial recovery needs a
reviewed missing-image repair or a new patch release; no blind overwrite/delete.
Moving tags across the two images are not atomic. This is an explicit operating
limit, not a code-merge blocker if the owner accepts new-patch recovery and pins
verified versions/digests until both images complete. It remains an activation
gate if the owner requires generic partial repair or concurrent external writers.

No tag/Release/image/deployment or environment mutation was performed. No merge,
workflow enablement, NAS/Compose/database/mobile/tm-sync release, or issue closure
was attempted. New-head CI and final PR review status are reported in PR #252.
