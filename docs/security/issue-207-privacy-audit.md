# Issue 207: pre-publication privacy audit

Audit date: 2026-09-30. Baseline: `origin/main` at `cfe8a1f`. **Publication gate: CLOSED.** This report is an inventory and remediation record, not approval to make the repository public.

## Method and coverage

- Scanned the committed tree (1,904 files) and reachable Git history with local Gitleaks 8.30.1. The history scan covered 710 non-merge commits and about 42 MB; the committed-tree scan covered about 18.8 MB. No repository content was sent to an external scanner. A separate working-directory scan included ignored and untracked local files, so its 73 results are excluded from publication findings; the tracked-tree scan had one result.
- Enumerated tracked and historical names for `.env`, CSV, JSON, XLSX, databases, dumps, logs, backups, archives, keys and certificates. No non-example `.env`, database, dump, log, archive, key or certificate was found in reachable committed history by filename. The historical `mobile/ios/.xcode.env` contains only the standard Node lookup.
- Searched tracked text for credential markers, personal paths, emails, private-network addresses, account identifiers and owner-specific repository links; checked importer fixtures, demo data, scripts, documentation and deployment configuration. Visually reviewed contact sheets of all 138 tracked PNGs and inspected their EXIF fields. The 14 images with EXIF carry only dimensions/color-space fields; no GPS, author or device identifier was found.
- This is a repository audit, not a verification of external credential status, GitHub unreachable objects, fork copies, release attachments or the owner's original broker exports.

## Findings

| Class | Location | Current/history | Finding and action |
| --- | --- | --- | --- |
| **BLOCKER** | `api/internal/importer/testdata/stonk-journal-trades-all-time-2026-07-11.csv` | History | A previously committed trade export with specific fills and timestamps was added in `6e09d51` and deleted in `6dacc1d`. Its independent synthetic provenance is unproved. Treat it as real-account data. Rewrite affected history in a controlled release operation, including other reachable branches/tags and remote copies, then verify the object is unreachable before publication. |
| **BLOCKER** | `api/internal/importer/testdata/tradermemos-export-journal.csv`, `tradermemos-export-unified.json` | Both before this patch; history afterward | The replacement CSV shares all 15 symbol/date pairs with the deleted export, so it cannot be accepted as independent synthetic data. Both fixtures and their fixture-dependent tests are removed from the current tree; a new invented two-trade test preserves stock/option multiplier, preview and P&L coverage. Rewrite their historical blobs with the export above. |
| **BLOCKER** | `api/internal/importer/testdata/sbi-*.csv`, `mt*-statement.*`; `api/testdata/generic_sample.csv`; `web/public/sample-*.{csv,json}`; `docs/demo/tradermemos-demo-trades.json` | Both | These static financial samples have no checked-in generator or independently verifiable provenance. Some contain explicit demo names, but a demo label or changed account number is insufficient under #207's rule. Replace with newly generated independent data, retain only the technical edge cases, run importer/Web checks, and include generator and provenance notes. Include old blobs in the history rewrite if owner origin cannot be ruled out. |
| **BLOCKER** | `docs/screenshots/**`, `marketing/public/screenshots/**`, `scripts/appstore/captures/**`, `scripts/appstore/out/**` | Both | Images display trades, balances and P&L. `README.md` documents `scripts/seed-demo.py` for the newer Web screenshots and `scripts/appstore/README.md` documents a demo capture, but some older captures depend on the unproven static demo book. Regenerate financial images from independently generated data and verify their inputs before publication; purge old images from history if derived from owner records. |
| **BLOCKER** | `docs/qa/issue-109/*.png`, `docs/validation/issue103/*.png` | Both | These show financial values or an export dialog without an explicit input-provenance record. Recreate from a generated test account or add verifiable generation evidence before publication. |
| **REVIEW** | `docs/superpowers/{plans,specs}/**` | Both | Upstream-authored planning docs contain a contributor's local home path and email. They appear to be upstream public attribution, not this fork owner's data. Confirm with the upstream owner before republishing; preserve intentional attribution, but replace incidental absolute paths if requested. |
| **REVIEW** | `marketing/next.config.mjs`; `docs/validation/issue{175,179,191}/README.md` | Both before this patch; history afterward | A machine-specific LAN origin and obsolete private-repository URLs were removed from the current tree. Historical copies remain and should be assessed during the controlled rewrite. |
| **SAFE / EXPECTED** | `api/internal/config/config_test.go` | Both | The sole Gitleaks hit is a fixed, human-readable test JWT value used in assertions (`generic-api-key` rule), not a deployable credential. No rotation is indicated by this hit. |
| **SAFE / EXPECTED** | `README.md`, `NOTICE`, `LICENSE`, current GitHub links; `.env.example` files | Current/history | Public author and fork attribution, documentation links and placeholder environment examples. Preserve attribution. Localhost/private-network examples with no owner-specific host are expected. |
| **SAFE / EXPECTED** | `docs/validation/issue{175,179,182,183}/**` | Current/history | Their README files record isolated throwaway APIs and deliberately seeded JPY/USD cases; screenshots visually match those small generated cases. This documents provenance for these particular validation images. |

No confirmed real credential was found in the reachable history, so this audit does not identify a specific credential to rotate. If the owner identifies a real value in a flagged file or an inaccessible historical object, rotate it **before** publication; deleting the file alone is insufficient.

## Remediation already applied in this branch

- Removed the current derived TraderMemos export fixtures and replaced their importer regression with invented data; the existing JSON unified-export test already covers the JSON format.
- Removed the owner-specific LAN origin and updated stale private-repository links in current documentation.
- Added ignore rules for local database variants, dumps, backups, archives and private-key formats. This cannot protect files already committed or prevent a forced add.

## Final release gate

1. Replace or remove every unresolved financial sample and screenshot above with independently generated data and recorded provenance; run affected tests and inspect the rendered replacements.
2. Review the original export privately with the owner. Plan and execute a coordinated history rewrite for sensitive historical blobs, including the named commits, branches and tags. Force-update remotes only in that controlled follow-up. Verify with a fresh clone, `git rev-list --objects --all`, a repeated Gitleaks scan and manual path/content checks.
3. Check forks, release assets, CI artifacts and caches separately; remove exposed copies where possible. Rotate any real credentials the owner identifies.
4. Repeat this audit against the exact commit and refs to be made public. Keep the repository private while any BLOCKER remains.
