# #244 cross-user isolation acceptance

Baseline: `005dd367879b1564e2add665a73dee3e0c26a333` (latest origin/main at start, including #272 / PR #285). Branch: `test/244-cross-user-isolation`. The requested `test/` prefix is an explicit task-specific exception to CONTRIBUTING's usual prefixes.

## Workspace protection

- INITIAL_BRANCH = `feat/272-sqlite-backups`
- INITIAL_HEAD = `a6511672266224dd5531a3f8c2acf7f2f6b8bd94`
- MAIN_SHA = `005dd367879b1564e2add665a73dee3e0c26a333`
- EXISTING_244_BRANCH = none (local and remote checked)
- EXISTING_244_PR = none (exact head and issue search checked)
- WORKSPACE_SAFE = YES

Development uses `/private/tmp/tradelens-244`. The owner's active checkout was not switched, stashed, reset or cleaned. Ignored personal reference files, `.env`, databases and dependencies stayed in the original checkout. No real financial, NAS, production or Render database was opened. All tests create disposable SQLite databases and temporary file/backup roots.

## Architecture and scope

`api/internal/api/cross_user_isolation_test.go` exercises Echo's production routes and JWT/PAT middleware with setup, registration and login through HTTP requests. A is the initial instance owner and B is an ordinary member. Both directions are tested on ordinary private routes: owner status grants no bypass there. Both users have two JPY accounts, distinguishable symbols, closed trades, open holdings, cash, journals, planned setup direction/targets/stops, tags, news/assets/predictions/evaluations, imports, image attachments/media, notification channels, share links and personal API tokens.

Fixtures assert different principals, owned-account IDs, authenticated PAT identities and owner reads before testing foreign IDs. Accounting uses the existing execution/import pipeline. FX, account-value bars, AI, IBKR and notification deliveries are mocked in memory or on test-only loopback servers; no external service credentials or live providers are required. IDs and bearer values come from the real API; deterministic sentinel data makes the assertions independent of those random identifiers.

Rejected writes compare hashes of **all application tables** and stored file contents. PAT last-used timestamps are normalized and usage-history rows are intentionally excluded from the database comparison because authentication records those even for denied requests; separate tests verify token ownership, listing, usage-history scoping, expiration, foreign revoke denial, and owner revocation. Response assertions never print full bearer secrets. Requests use safe synthetic secrets, and the primary server logger discards request logging.

Analytics tests first save A's responses, then create B's distinctive dataset and assert A's results remain equal. This catches numeric aggregation leaks, not just leaked names. Account filters include no filter, own, foreign, missing, comma-separated and repeated mixed selections. Additional cases cover currency conversion, dates, symbols, search, sorting and pagination parameters. Unsupported search/sorting/pagination parameters are tested for isolation, not falsely reported as implemented filtering. Breakdown includes symbol, setup, tag, mistake and quality. Holdings and Home/report data are composed from trades/account-value/analytics; there are no separate Home/positions endpoints.

## Route inventory and test matrix

See [every registered method/route](endpoints.md). There are 150 routes: 134 ownership boundaries COVERED, 10 INTENTIONALLY PUBLIC and 6 NOT APPLICABLE to private-object isolation. No current route is NOT COVERED or PARTIALLY COVERED at the ownership-boundary level. This does **not** mean every functional combination of every endpoint has been tested.

`TestCrossUserIsolationRouteInventory` checks the documentation against the actual registered routes. Adding a route requires an inventory update and an ownership audit. Every private route is separately driven anonymously and must return 401.

| Resource | Owner | Foreign JWT / PAT | Anonymous | Evidence |
|---|---|---|---|---|
| Accounts, account settings, execution and cash writes | API creation/update/delete/read-back | Known IDs denied; unchanged rows/files; nested parent/child checks | 401 | IDOR, OwnerMutations, fixture checks |
| Trades, journals, tags, setups, direction, targets, stops, risk | Owned reads and changes; unrelated fields preserved | Known IDs and foreign child associations rejected | 401 | IDOR, SelfSettings, OwnerMutations |
| Analytics, account value, open holdings, setup/tag breakdown | Nonempty owner responses and before/after B comparison | Scoped empty result or generic scope rejection; mixed accounts never aggregate B | 401 | AggregatesAndLists |
| News, assets, predictions, evaluations and research notes | Creation, evaluation, reads, updates/deletes; existing reviewed-AI owner tests | Foreign and mismatched nested IDs denied; no changes | 401 | IDOR, OwnerMutations, existing news tests |
| Attachments/media | Upload, stream, delete | Foreign upload/read/delete, guessed key/filename/traversal denied; file hashes unchanged | 401 | IDOR, FilesImportsExports |
| Imports | CSV preview/fresh commit; synthetic statement commit; legacy batch commit | Foreign account/batch denied; client parent cannot redirect owned batch | 401 | FilesImportsExports, ConfiguredServices |
| JSON, CSV, ZIP and news exports | Owner sentinel present; ZIP entries decompressed and inspected | Foreign account/news selection denied; no foreign sentinel/ID/content; no generated files | 401 | FilesImportsExports |
| API tokens | Identity, list, usage history, expiration, revoke | No foreign history/secrets, foreign revoke denied; PAT matches JWT private-route matrix | 401 | IDOR, TokensAndSharing |
| User settings and credentials | Distinct preferences/checklist/risk/goal/alert settings, password and TOTP flow | Body/query user_id cannot select B; B read-back preserved | 401 | SelfSettings, Credentials, ExcursionAndGoalDelete |
| Notifications | Mock webhook actually receives owner test; push/event fixtures | Foreign channel test never sends; event lists scoped; foreign push unregister is a no-op | 401 | NotificationDelivery, IDOR |
| AI review and Flex Sync | Enabled mock providers execute owner operations; stored reviews read back | No provider request or DB write for foreign resources | 401 | ConfiguredServices |
| Excursion | Supported historical owner computation persists MAE/MFE | Known trade rejected under JWT/PAT, no DB mutation | 401 | ExcursionAndGoalDelete, IDOR |
| Instance AI settings | Owner management and existing mock-provider tests | Member blocked before singleton config access or provider connection | 401 | InstanceSettings |
| SQLite backups / administration | Owner snapshot and status; existing admin owner tests | Member JWT/PAT blocked; guessed downloads unavailable; snapshot bytes unchanged | 401 | AdminBackupAndAnonymousRoutes; existing #272 suite |
| Public sharing | Scoped bearer aggregate; default amount redaction | Authenticated management remains user-scoped; public query cannot broaden capability scope | Deliberately public; invalid/revoked/expired = same 404 | TokensAndSharing; existing share tests |

Existing owner tests are retained: `handlers_test.go`, `execution_handlers_test.go`, `trade_journal_test.go`, `trade_delete_test.go`, `setup_handlers_test.go`, `attachment_handlers_test.go`, `preferences_handlers_test.go`, `me_handlers_test.go`, `totp_handlers_test.go`, `settings_handlers_test.go`, `coach_settings_handlers_test.go`, `coach_handlers_test.go`, `coach_stream_handler_test.go`, news/prediction/export tests, API token tests, import tests, currency/filter/account-value tests, `admin_handlers_test.go`, `backup_handlers_test.go`, and `internal/backup/backup_test.go`. The full API test run includes these rather than only the new security matrix.

## Findings and focused fixes

1. **HIGH — missing owner boundary on instance AI configuration.** OCR and Coach configuration use singleton rows (`id=1`), not per-user settings. Authenticated ordinary members previously had access to configuration management. Keep the existing instance architecture and add the existing `requireAdmin` middleware to all eight configuration read/write/test/model routes. Owner behavior remains; ordinary private trade/news/OCR consumers continue using the instance configuration. No migration, new RBAC, per-user AI settings or secret rotation is introduced. `TestCrossUserIsolationInstanceSettings` failed before the guard and passes after it. Member Settings clients must now handle 403 for these owner-managed sections; Web permission UX is not validated here.
2. **MEDIUM — rejected journal PATCH could partially persist.** A PATCH containing a journal update plus a foreign tag returned 400 after writing the journal. Move tag ownership validation before any journal/link mutation. The regression compares complete persisted-row hashes and fails on the baseline. Owner partial edits retain existing targets, risk, setup, fills and accounting fields.

No foreign account/trade/news/attachment content leak was reproduced in the tested private routes. Safe existing conventions are preserved: foreign token usage history is `200 []`, unknown account scopes may return generic `400 unknown_currency`, account-value and cash selection can ignore foreign accounts, and foreign regroup requests return generic 500 without mutation. These are not reported as successful access. Missing and foreign resource errors are compared where applicable.

## Backup architecture

SQLite backups intentionally contain the **whole instance**, including all users; they are not per-user exports. Admin authorization is re-read from the database. The API exposes status and run only, no backup download or restore route. Request-supplied destination/retention values cannot alter the configured backup root. The existing #272 tests verify per-instance directories, retention leaving other instance files intact, restart identity, concurrent locks/WAL, failure cleanup, and disabled demo behavior. Both these tests and the new owner/member tests run in the explicit CI gate. No live restore or NAS test is performed.

## #273 / PR #292

PR #292 (`feat/273-review-inbox`) remained OPEN and unmerged when the baseline and final delivery were checked. No Cloud branch or unfinished Review Inbox code was changed/copied. Review Inbox is NOT APPLICABLE to this baseline.

Before enabling it, reuse the two-user fixture for known foreign account IDs, backlog/preferences self-scoping, mixed selections, list/count/aggregation, filter/search/pagination, child transitions and rejected-write read-back. Update the route inventory and run the security gate on the merged SHA. Keep Beta blocked if a new high-risk route has not been verified. Coordination is through the reusable fixture and these follow-up integration checks; the Cloud work remains untouched.

## Validation and release assessment

Commands (from `api/`):

```sh
go test ./internal/api ./internal/backup -run 'TestCrossUserIsolation|TestBackup|TestDemoBackup|TestInstanceIsolation|TestRetention' -count=1
go test ./internal/api -run TestCrossUserIsolation -count=3
go test ./...
go test -race ./...
go vet ./...
```

From the repository root: `git diff --check`.

CI now has explicit offline isolation and targeted race steps in `.github/workflows/api-ci.yml`, and retains the ordinary full API tests/vet/build. Local command results and final-head CI evidence are recorded in [results](results.md).

The API isolation gate can be accepted only after local checks and final-head API CI pass, with no unresolved high/critical isolation finding. This report does not authorize Private Beta registration, a release, a merge or deployment. Draft PR delivery is intentional; owner/security review is still required.

## Remaining limits

- Coverage means the tested API ownership boundary, not exhaustive combinations, a penetration test, timing/side-channel proof, every error or database failure, or concurrent user deletion during an in-flight operation.
- SQLite and local-disk storage are tested. PostgreSQL, alternate blob stores, reverse-proxy misconfiguration and publicly exposing raw storage directories are not validated.
- External Yahoo, broker, AI and webhook services are replaced by mocks. Their availability, vendor privacy and accuracy are outside this acceptance.
- No Web code changed. Web permission UX after the owner-only AI configuration change is unverified; mobile is not validated and is outside current fork scope.
- Existing instance owners intentionally administer users and backups. They are trusted operators; the suite does not claim owner protection against filesystem/database administrators.
- No invitation/registration redesign, billing, mobile implementation, accounting reinterpretation, dependency upgrade, live restore, release or Render action.
