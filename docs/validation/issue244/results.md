# Validation results

Local macOS arm64, Go **1.27.0** (module-selected toolchain), disposable SQLite and temporary storage. Recorded 2026-10-10, Asia/Shanghai.

| Check | Result | Evidence |
|---|---|---|
| Focused isolation + backup gate | PASS | [security-gate.txt](evidence/security-gate.txt) |
| Final isolation suite, count=3 | PASS | [repeat.txt](evidence/repeat.txt) |
| Final `go test ./...` | PASS | [go-test.txt](evidence/go-test.txt) |
| `go test -race ./...` | PASS | [go-race.txt](evidence/go-race.txt) |
| Final security + backup subset with `-race` | PASS | [target-race.txt](evidence/target-race.txt) |
| Final `go vet ./...` | PASS, exit 0 | [go-vet.txt](evidence/go-vet.txt), empty successful output |
| `git diff --check` | PASS | Run before commit |

The full race API package took 380.279 seconds. It ran on the same production fixes, before the last harness-only strengthening (normalizing token usage timestamps while comparing token metadata, exact generic scope errors, multi-own-account selections and explicit owner regroup). The final security subset was rerun with race after those changes and passed in 66.017 seconds. Final full ordinary tests and vet also passed. No unsupported race fallback was needed.

Regression evidence before fixes:

- The instance-settings test expected member 403 and received 200 on the baseline.
- The journal + foreign-tag test received the expected 400, but the complete database-state comparison detected a changed `trade_journal` row.
- Both now pass; owner operations remain covered. Baseline failure logs are deliberately not published as detailed exploit transcripts.

Remote evidence is the Draft PR's checks on its exact final head SHA, linked in the PR body and final delivery report. The required `Test API` job includes explicit isolation/race steps and full API tests/vet/build. A pending/failed final-head check must not be inferred from these local passes.

## Delivery assessment

- Tested branch: no unresolved high/critical cross-user isolation finding reproduced; 134 current private/instance ownership boundaries exercised.
- READY_FOR_REVIEW: YES once final-head API CI passes; PR remains Draft as requested.
- READY_FOR_MERGE: NO, pending independent owner/security review and explicit merge authorization.
- PRIVATE_BETA_ISOLATION_READY: NO for current main/deployment. The confirmed high-severity instance-settings defect is fixed only in this unmerged PR. Approve/merge and verify the deployment source before treating a running instance as ready.
- Web handling of member 403 in instance AI settings is unverified. PostgreSQL, alternate storage, live external providers, mobile, production/NAS restore and Render are outside this validation.
- #273/PR #292 remains unmerged. Its future endpoints need the documented integration checks before enabling them.
