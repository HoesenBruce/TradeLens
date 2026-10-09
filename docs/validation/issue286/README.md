# #286 implementation validation — 2026-10-09

Scope: offline implementation and Draft PR only. Owner explicitly declined any
live Render deployment/configuration changes until secrets setup and later
controlled acceptance. No dashboard screenshots are captured or uploaded.

Base source: main `11427daad2956847a5735bc2cc96e2408b13e163`.
Branch: `chore/286-render-release-deploy`.

## Read-only release-chain audit

- `release-please.yml` and `docker-publish.yml`: active in GitHub on 2026-10-09.
- Release Please main run `37901765805`: succeeded, no release/publication created.
- Latest stable Release: v0.2.1; successful standalone publisher `37590176429`.
- `ghcr` environment: owner reviewer, administrator bypass disabled, main/tag
  selected-branch policy exists. Recheck exact selected rules before activation.
- Repository secret name inventory contains RELEASE_PLEASE_TOKEN. No secret
  values were read. No Render credentials were configured.
- Render branch/Free/Auto-Deploy settings were **not live audited**; the code
  checks them through authenticated service GET before any deployment.

## Local verification

- 10 unittest methods passed with multiple subcases: stable/prerelease/draft/tag
  validation, old/divergent Live rejection, explicit older-version bootstrap,
  successful two-image evidence/registry checks, missing/failed/mismatched
  publication, same-parent in-progress automatic call and recovery from a Demo-only parent failure, full Mock Render/public
  HTTP flow, exact commitId POST, credential separation, failure/timeout/mismatch,
  failed smoke, sanitized failure record, workflow integration guards.
- Existing Docker release metadata regression: 3 unittest methods passed.
- Actionlint passed for all four changed workflow files.
- `git diff --check` passed.
- The new smoke function passed against a disposable local container using the
  existing `tradelens-demo:243-sync` image, limited to 512 MiB / 0.5 CPU, on
  loopback port 19086. Container removed after verification. This reused fixture
  image is **not a newly built issue286 image or a public Render deployment**.
- Same local smoke confirmed /demo, login, protected unauthenticated read denial,
  non-admin JPY account, 107 closed trades, 214 executions, P&L 25,000,
  contributed capital 950,000, account value 975,000, news/calendar, explicit
  demo_read_only responses and unchanged trade data.

## Outstanding acceptance

Keep #286 open and PR Draft. No deployment version/SHA/deploy ID/timestamps exist
for this change because no live deployment was authorized or attempted.

Owner must configure the render-demo environment/secrets/review policy, approve
and publish a stable release containing Demo + smoke contract, then authorize one
main-based manual controlled deployment. v0.2.1 lacks those files and is rejected.
Record actual Free/no-disk/Auto-Deploy-Off service read-back, exact Live SHA and
public smoke/Web acceptance in text. Only then opt in via
DEMO_AUTO_DEPLOY_ENABLED=true and verify a legitimate automatic release chain.

GitHub nested-workflow/runtime approval behavior, live artifact download/registry
probes, authenticated Render response shape, API completion and public smoke
remain unverified. No API/Web application code changed; full application suites,
new image rebuild, UI browser acceptance and mobile runs were outside this
workflow-only implementation validation. Remote PR CI is reported separately.
