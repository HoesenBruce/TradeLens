# Final acceptance preflight and Render configuration — 2026-10-09

Repository: HoesenBruce/TradeLens. PR: https://github.com/HoesenBruce/TradeLens/pull/264.
Issue: https://github.com/HoesenBruce/TradeLens/issues/243.
Public URL: https://tradelens-demo.onrender.com.

## Execution boundary

This file records the initial preflight and subsequent owner confirmation.
For current synchronization/validation status, see sync-main-acceptance.md and
PR #264's latest text-only validation comment. Gates A–C passed; the branch has
since been rebased without conflicts. Final remote CI and the required affected
public retest govern merge. No Render settings change or issue closure occurred.

## Read-only preflight

- PR #264: OPEN, unmerged, Draft; head branch `feat/243-render-demo`.
- Head and fetched remote branch: `b5ecb1f2960d1f52e8f58ae1d7d630a46d2fbbe8`.
- GitHub PR baseRefOid response: `e214637e0d73949abe6f26277574e5278c5b8490`.
- Current main branch API and fetched origin/main: `51118ea03ca565024a7e3130cedb28faae2bdc3b`.
  GitHub's PR baseRefOid response differs from the current branch ref; it must
  not be used as proof of the current main tip.
- Five PR commits cover isolated demo implementation and validation evidence;
  diff: 54 files, 1,562 insertions, 14 deletions, plus existing image evidence.
  Normal self-hosted Dockerfiles, Compose and GHCR workflows are outside the diff.
- GitHub mergeable: MERGEABLE; merge state: BEHIND. No reported conflict.
- `/private/tmp/tradelens-243` was clean before this report was added.
  The primary checkout is on `feat/272-sqlite-backups` with unrelated changes;
  those changes were left untouched.

## Owner-confirmed Dashboard configuration

The following observations were supplied by the owner, not independently verified
by Codex:

| Setting | Owner-confirmed observation |
| --- | --- |
| Service | tradelens-demo |
| Runtime | Docker |
| Instance type | Free |
| Workspace plan | Hobby |
| Payment method | No card on file |
| Service status | Live / Deployed |
| Active services | 1; no other services visible in workspace overview |
| Auto-Deploy | Off |
| Docker command override | Empty |
| Current unbilled charges | USD 0.00 |
| Projected monthly charges | USD 0.00 |
| Deployment branch before merge | feat/243-render-demo |
| Public URL | https://tradelens-demo.onrender.com |

No paid resource was identified in the supplied observations. That does not
independently establish the disk configuration or full resource inventory.

- Persistent Disk: **absent, owner-confirmed on 2026-10-09**.
- Actual live deployment SHA: **`3e8f7b8090673ba3d23663f2d47eecfbf590dc86`, owner-supplied current Live deployment SOURCE link**. These are owner observations, not independent Codex Dashboard verification.
- Render management access: built-in browser reached the Sign In page; no
  authenticated management connector was available. No login credentials were
  requested, entered, or recorded.

The versioned render.yaml declares one Free service with Auto-Deploy off and no
disk declaration. This is intended configuration, not evidence of live settings.

## Deployment identity assessment

The existing public-new-acceptance.md identifies target runtime revision
`3e8f7b8090673ba3d23663f2d47eecfbf590dc86`; it explicitly states that public
system info exposes a version, not a Git SHA. Its assertions include 107 trades,
JPY 25,000 realized net P&L, JPY 975,000 closing account value, fictional calendar
filtering, write denials, 971.49 seconds idle and 22.82 seconds readiness recovery.
The old unexpired refresh token was rejected after regeneration.

The only subsequent PR commit is `b5ecb1f2960d1f52e8f58ae1d7d630a46d2fbbe8`.
Its changes are restricted to docs/demo-render.md and validation evidence.
No runtime-code change follows the acceptance target within this PR. Deploying
that documentation-only commit is not necessary merely to update the evidence.
The owner subsequently supplied the Live deployment SOURCE link to that exact
full SHA. Deployment identity is consistent with the public acceptance target.
Gate B passes based on the supplied owner configuration and identity evidence.

## CI snapshot and required policy

The inspected head had successful Test API, Test web and Conventional PR title
checks. These are a preflight snapshot, not final Gate D approval.
The main ruleset requires those three checks, a pull request, linear history,
and strict up-to-date checks. PR #264 is BEHIND; synchronize with current main
and rerun required checks before a controlled merge. Do not bypass the ruleset.

Gate C PASS: reviewed the current public acceptance report and sanitized evidence;
no unresolved security or functional blocker was identified. Gate D BLOCKED:
the current head checks passed, but the branch is BEHIND under strict required
checks. Following the task stop rule, no branch update or merge was performed. Gate E and post-merge smoke
were NOT RUN. PR remains Draft/OPEN; issue #243 remains OPEN. No merge SHA exists.

## Exact owner action needed

Disk absence and the current Live SOURCE SHA were supplied by the owner.
The next required remediation is synchronizing the PR branch with current main
and obtaining successful required checks on that new head. Current green checks
do not establish compatibility with the current main tip.

With Gates A–C passed, synchronize the PR branch, pass CI
on the final head, commit necessary text-only records, recheck CI, mark Ready and
squash merge. Then switch Render to main with Auto-Deploy Off, manually deploy
the intended merged SHA, verify Live SHA/configuration and public smoke, and only
then close #243. If management access remains unavailable, the owner must perform
the Render branch change/manual deployment and supply textual confirmation.

## Proposed future issue specification (not created)

Title: `ci: deploy Render demo automatically after stable releases`

Trigger only for published stable GitHub Releases; reject prereleases and draft
releases. Resolve and verify the release tag and exact source SHA. Keep Render
Auto-Deploy Off, deploy the exact release commit, serialize deployments to prevent
overlap, wait for completion with a bounded timeout, and verify the deployed SHA
and public health. Run minimal demo login/read-only/fictional-data smoke checks.
Preserve Free instance/no Persistent Disk/no additional paid resources. Store
credentials in protected secrets, never expose them in logs or evidence, and
report failed deployment or smoke checks clearly. Acceptance requires exact SHA
agreement and successful health/smoke; failed runs must not report completion.
No release deployment workflow or follow-up issue was created in this task.
