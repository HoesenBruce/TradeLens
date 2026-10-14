# PR #306 cloud remediation

## Scope and baseline

Repository `HoesenBruce/TradeLens`, existing Draft branch
`fix/299-legacy-coach-isolation`. Fetched baseline main:
`021ffa091ca9a0e4c29a3d5acdffeb12d54a2f01`; starting PR head:
`df20e02bd7d8cde7f00d342aac418d55545ed5fd`. GitHub reported the head repository
and author as HoesenBruce and repository push/admin permissions before editing.
No local developer worktree, database, reviewer harness or test result was reused.
The API binary, SQLite database, users, providers and browser fixtures were created
in this cloud task. The API production tree is unchanged by this remediation.

## Root causes and fix

The application-wide QueryClient survived authentication transitions. OCR and
unused legacy Coach settings had identity-independent keys, and successful old
mutations could write those keys after a new login. OCR forms retained dirty local
state; global drawer state could retain private edit snapshots. Model discovery
persisted private provider URLs/model names without identity information.

`SessionQueryProvider` now creates a new QueryClient and mounted child tree for
each existing auth session counter. It clears the retired query/mutation cache,
including cancelling query observers. HTTP responses and refresh adoption are
additionally guarded by a credential revision: late old-session results are
rejected with AbortError, cannot refresh old credentials, retry using a new user's
credentials or sign out a replacement identity. Normal same-session automatic
refresh preserves the session boundary. Explicit password token replacement uses
signIn and therefore retires old state. Login, logout and expiry also clear model
lists and private drawer snapshots. No page reload is needed for this isolation.

OCR/Coach query and mutation keys include session; queries require authenticated
`/me.is_admin`. Permission denial and query errors hide cached settings. Save
callbacks cannot populate another session's key. OCR forms are keyed by session;
late save/test/model callbacks cannot restore a draft or update model discovery
in a subsequent session. Admin user query keys are session-scoped too. The broader
QueryClient boundary covers other private hooks, including alerts, tokens,
account/integration status, risk rules and journal data, without rewriting them.

There is no Web React Query persistence/hydration provider. The model-discovery
store now stays in memory; startup removes the old localStorage entry and does
not hydrate it. Ordinary display preferences and public assets are preserved.

The follow-up worker policy makes all authenticated requests network-only before
any generic asset/API cache handler, regardless of API origin, proxy prefix or
worker scope. A shared path policy recognizes `/api/` as a path segment, including
`/gateway/api/v1`, and retains the existing confidential diagnostic/export path
rules. Anonymous recognized API paths are network-only too. The Web SPA currently
uses Vite's root base and root worker scope; configurable API bases are independent
of that scope. A custom API base without `/api/` is covered by its Authorization
header. This does not add subpath SPA hosting or change routing/CORS contracts.

Activation deletes the retired `api-cache`, `apis` and `others` caches. The latter
two could contain prefixed or custom-base API bytes from Serwist's default handlers.
Across all remaining caches it removes recognized API entries, authenticated
entries and old `__tm_auth_scope` entries. The same policy drives network handling
and migration. Public static/precache resources and app-shell navigation remain
cached; live private API reads now require network, even for the same bearer.
React Query still retains authorized in-memory data within the current session
and resets at auth changes. The worker does not turn that state into a persistent
offline API store. Disconnected clients still require reconnection for the updated
bundle/worker; previously exported files or arbitrary browser-storage access are
outside this request/UI boundary.

The new PO entries contained correct translations but lacked
`#. js-lingui-explicit-id`. Lingui therefore compiled hashed message-text keys
instead of `coach.legacyUnavailable` and `news.analysisUnavailable`. Added that
marker in all five source catalogs and regenerated the tracked TS catalogs.
Catalog comparison confirmed every existing shared key/value is unchanged: only
the two wrongly hashed keys become the two intended explicit keys per locale.
No localization architecture changes or legacy generation re-enablement.

## Permanent regressions

- `web/src/components/SessionQueryProvider.test.tsx`: mounted dirty state,
  owner/member/other owner/login/logout, query/mutation disposal, late read/save,
  model/drawer state reset, expired authentication and a background 403.
- `web/src/lib/api/client.test.ts`: old refresh cannot replace new credentials
  or sign out the new identity; existing refresh/auth tests remain intact.
- `web/src/i18n/unavailable-catalogs.test.ts`: compiled entries in all five locales.
- `web/e2e/session-isolation.spec.ts`: actual production Chromium, real disposable
  API login/logout and authorization, online/offline owner/member/owner x3,
  delayed old owner response x3, owner A/B and dirty/reopened forms, old model
  persistence, reload/hydration, five narrow production locales,
  network-only API behavior, manual News cancel/re-entry/save/reload.
- `web/e2e/coach-isolation.spec.ts`: extended existing upgrade/purge/offline
  regression to include old unscoped `/me` and `/admin/users` entries.
- `web/e2e/session-isolation.config.ts`, `worker-regression.config.ts` and
  `web/scripts/serve-security-build.mjs`: deterministic production harness using
  installed Chromium, static dist and a proxy to the disposable API. Only the
  eagerly mounted unrelated FX calculator uses a synthetic rate response.

## Prefix API follow-up regressions

The independent review of `6210e3e844ee8d124d01f95fdb6253c55fba4e2a`
found two blocking defects: path-prefixed API calls fell through to unscoped
Serwist caches, and activation omitted old prefixed API entries. That head's
historical passing default-path results below do not certify prefix isolation.

The login/settings API-server input also preserves a full prefixed base instead
of stripping `/api/v1` from a path that normalization cannot reconstruct. Bare
origin input behavior stays unchanged. Client round-trip tests and the real
prefixed login/logout UI cases protect this supported deployment configuration.

Permanent follow-up coverage:

- `web/src/lib/workerCachePolicy.test.ts`: segment matching, public-path exclusions,
  authenticated custom/cross-origin URLs and retired scoped keys.
- `web/e2e/worker-api-isolation.spec.ts`: `PREFIX_BEARER_X3` for default, prefixed,
  custom API bases and a narrower worker scope, using fresh Chromium contexts.
  Real disposable API owner/member authorization remains 200/403 online; both
  identities' private requests fail offline and create no persistent API entries.
- `PREFIX_CACHE_MIGRATION`: seed a disposable legacy worker in four cache names,
  prove pre-upgrade replay with a boolean assertion, then verify automatic upgrade
  deletes default/prefixed/authenticated/scoped entries. Private offline requests
  fail and a public precached favicon continues to work. Repeated three times.
- `web/e2e/security.config.ts`: the existing production session/localization suite,
  prefixed Owner → Member online/offline UI transitions three times, and worker
  upgrade/isolation/migration three times. Existing UI fixtures remain synthetic.
- `web/scripts/run-security-e2e.mjs`: builds a local API, creates and deletes its
  SQLite database, runs local synthetic market fixtures and shuts down all owned
  services. Refuses to reuse an existing API. No production DB/provider is used.
- `.github/workflows/browser-security-ci.yml`: runs the actual production bundle
  and `Chromium security E2E` on PR heads, alongside API/Web/title checks.

Run from `web/` after a production build:

```sh
node scripts/run-security-e2e.mjs
```

The harness uses `go` from PATH; `E2E_GO_COMMAND` can select an installed Go 1.27
binary. Chromium uses an explicit `E2E_CHROMIUM_PATH`, installed `/usr/bin/chromium`,
or Playwright's installed Chromium. Final validation counts and exact-head CI URLs
belong in PR #306's current acceptance evidence. Keep Draft for independent review;
this implementation task does not authorize merging or release.

## Historical remediation evidence (before prefix follow-up)

The archived starting head was separately production-built in cloud. Running the
new principal browser regression against that bundle failed with only the boolean
`private input present = true` after member login. The English unavailable-state
assertion also failed, showing the internal ID. No fixture configuration or raw
private response was printed. The fixed build passed all 14 principal Chromium
cases; all three online/offline and delayed-response repetitions passed. The
existing worker-upgrade regression passed with the expanded purge assertions.
All five locale cases verified the exact translated Coach and News explanations,
absence of internal IDs, zero console/page errors and no narrow-screen horizontal
overflow; screenshots were captured without private settings fields.

Full Go test/vet/race completed successfully on the unchanged API production tree;
the API race package took 513.125s. Full Web suite passed 192 files / 1100 tests
before the final two regressions were added; final-head counts and exact CI links
are recorded in PR #306's description after the final validation run. Typecheck,
check and production build passed; pre-existing lint warnings remain.

## Reproduce in Cloud

Use installed Go 1.27, Node 24, pnpm 11 and Chromium. Keep fixture databases outside
the repository; never point this suite at production. Start a local Generic Bars
fixture on port 8100 that returns the requested symbol/interval with an empty
`bars` array, timezone America/New_York and adjustment_status unadjusted. Start
a freshly built `api/cmd/server` with:

```sh
TM_HTTP_PORT=8099 TM_DB_PATH=<disposable-directory>/qa.db \
TM_JWT_SECRET=<synthetic-secret-at-least-32-characters> TM_JOBS_ENABLED=false \
TM_MARKET_DATA_PROVIDER=http TM_MARKET_DATA_HTTP_BASE_URL=http://127.0.0.1:8100 \
TM_CORS_ORIGINS=http://localhost:5189,http://127.0.0.1:5189 <disposable-server-binary>
```

The suite bootstraps only the synthetic `.invalid` identities. For repeated runs,
it reuses those synthetic identities on the same disposable API, and adds fresh
News/account/trade fixtures. Stop the test servers and discard the DB afterward.

From `api/`:

```sh
go test ./... -count=1
go vet ./...
go test -race ./... -count=1
```

From `web/`:

```sh
pnpm run i18n:extract
pnpm run i18n:compile
./node_modules/.bin/vp test run --maxWorkers=2
./node_modules/.bin/tsc -b --noEmit
./node_modules/.bin/vp check
./node_modules/.bin/vp build
./node_modules/.bin/playwright test -c e2e/session-isolation.config.ts
./node_modules/.bin/playwright test -c e2e/worker-regression.config.ts --grep 'security worker'
```

Lingui extraction was inspected but its unrelated catalog reordering and
pre-existing translation gaps were not committed. Source translations for these
two required IDs are complete. Two consecutive compile runs must produce identical
TS hashes. If Cloud pnpm attempts to reinstall dependencies despite the installed
toolchain, use `pnpm --config.verify-deps-before-run=false run i18n:compile` (or the
same installed Lingui CLI); dependency versions/lockfiles are unchanged.

From the repository root:

```sh
git diff --check origin/main...HEAD
```

## Boundaries and next action

Offline login is not a supported product workflow. The closest real workflow is
warmed owner logout offline, reconnect/login member, then member settings offline;
member offline reload and navigation are also exercised. The delayed-response
case disables worker registration only to control delivery of a real API response;
worker replay/upgrade and normal offline cases run the actual production worker.
Owner configuration is server-wide and readable by every authorized owner; the
A/B test changes the authorized baseline between logins and proves the previous
owner's unsaved draft is not reused. It does not invent per-owner server settings.

Core/API coverage includes legacy Coach JSON/SSE/history/zero-provider/write
containment, JWT/PAT/ownership, preserved history, OCR provider contracts, SBI,
valuation, journal/plans, manual News/predictions and SQLite backup. Real external
vendors, Postgres, mobile and production deployments are outside this validation.
Existing old clients require reconnection to receive the fixed worker/bundle.
History and operator backups are not rewritten. Existing broader non-required
translation gaps remain. Review Inbox #292 is not integrated.

Keep #306 Draft for a fresh independent security review of the revised head.
No merge, release, tag, deployment, other PR mutation or issue closure is part of
this task. Exact final head, validation counts and its CI links belong to the PR
metadata and final task report so this file need not embed its own commit hash.
