# Agent rules

## Git workflow guardrail

Follow the canonical Git workflow in `CONTRIBUTING.md`.

For development work:

- never implement feature work directly on `main`;
- keep `main` runnable;
- create a short-lived branch using the repository naming convention;
- use one branch / pull request for one coherent change by default;
- prefer separate Git worktrees for concurrently developed issues;
- do not introduce a long-lived `develop` branch unless the repository policy is explicitly
  changed.

Allowed branch prefixes:

- `feat/*` — new features;
- `fix/*` — bug fixes;
- `refactor/*` — refactoring;
- `docs/*` — documentation;
- `chore/*` — tooling, CI, dependencies, build, and maintenance.

When an issue exists, prefer including its number in the branch name.

## Private-fork platform policy

This repository currently follows a **Web-first, mobile-preserving** development policy.

The Web application is the active product and acceptance target for fork-specific feature work.
The existing Expo mobile application remains in the repository, but iOS and Android are not
currently required delivery targets unless an issue explicitly opts them back in.

For ordinary fork development:

- implement and verify the Web experience when UI work is requested;
- keep shared API, domain, data-model, persistence, importer, market-data, analytics, and service
  changes reasonably platform-neutral;
- reuse existing shared abstractions instead of introducing Web-only assumptions without need;
- do not remove, rewrite, or intentionally break mobile code merely because mobile is currently
  outside active validation scope;
- do not add iOS/Android implementation or validation work to an issue by default;
- mobile-specific implementation is optional unless an issue explicitly requires it;
- Android emulator / real-device and iOS simulator / real-device validation are **not required**
  to complete a Web-focused issue;
- preserving mobile-compatible contracts does **not** mean mobile behavior has been tested;
- if a change is known to leave mobile unsupported or unverified, document that limitation in
  the issue or PR rather than expanding scope automatically;
- when mobile support is reactivated, treat the existing mobile codebase as the starting point:
  run compatibility testing, fix regressions, and add platform-specific adaptations as needed
  rather than rebuilding the client from scratch by default.

See `docs/FORK_DEVELOPMENT.md` for the canonical scope and upstream-compatibility policy.


## Broker/accounting change guardrail

For importer, trade-grouping, position, or P&L work, preserve upstream/default behavior for
non-target brokers unless the issue explicitly authorizes a semantic change.

Before changing shared accounting code:

- identify the exact broker/source + execution/position semantic that needs special handling;
- keep unknown and non-opted-in brokers on the existing upstream/default path;
- do not select accounting rules from account `cash`/`margin` labels or capabilities alone;
- do not assume an SBI rule applies to IBKR, other non-Japanese brokers, or every Japanese broker;
- prefer an isolated broker-specific strategy/resolver seam over a global engine rewrite;
- add regression tests for representative non-target broker behavior when shared code is touched;
- preserve existing default partial-close `trade.net_pnl` semantics unless the issue explicitly
  changes them;
- do not fix unrelated inherited frontend/backend P&L differences as incidental work.

Any intentional change to an existing broker's accounting semantics needs explicit issue scope,
before/after examples, and a note about historical-statistics/recalculation impact.

The canonical policy and examples are in `docs/FORK_DEVELOPMENT.md`.

## Web UI features ship only after a full end-to-end run

**Any new or changed Web UI must be driven end to end before the work is called done or a PR
is opened. A feature with an unexercised required Web case is unfinished, not "probably fine".**

Static checks and screenshots are useful but do not replace exercising the behavior.

### Browser preference

Use the Codex built-in browser (`@Browser`) by default for Web interaction and end-to-end
acceptance. Use an external browser only when the built-in browser is unavailable or cannot
complete a required verification, or when the user explicitly requests one. Explain the reason
when falling back to an external browser.

### The cases that must pass when applicable

Run the relevant cases and keep screenshots or other evidence where useful:

1. **Happy path** — the thing does what it says.
2. **The inverse** — a filter that *excludes*, a toggle turned off, an empty result. A
   change that leaves the screen looking identical proves nothing.
3. **Read-back** — reopen the control and confirm it reflects the applied state, in the
   summary, label, or checkmark the user reads it from.
4. **Re-entry** — reopen the surface and confirm it starts from what is in force, not
   from what a previous visit was abandoned on.
5. **Cancel** — dismiss without committing; the app must land back where it started.
6. **Reset** — the clear/reset path returns to the baseline.
7. **Every state transition the feature has.** Both directions of a toggle, both
   directions of a transition. Arriving somewhere is not proof you can get back.

Only apply cases that exist for the feature. Do not invent mobile validation work for a Web-only
issue.

### Screenshots and visual verification

For visible Web changes, inspect the rendered result rather than relying only on DOM,
accessibility-tree, type-check, or lint output. Keep screenshots for important states when they
materially help review.

### Point the app at a real API first

Before trusting a Web result, confirm the app is pointed at a working API and is not displaying
stale cached data.

When the dev server's data is thin or its state is unknown, stand up a throwaway one from the
working tree rather than testing against a moving target:

```
TM_HTTP_PORT=8091 TM_DB_PATH=<tmp>/qa.db TM_JWT_SECRET=$(openssl rand -hex 32) go run ./cmd/server
POST /api/v1/setup      # first user
POST /api/v1/accounts   # an account
POST /api/v1/executions # buy + sell pairs, dated to straddle whatever is being tested
```

Seed data that makes the assertion sharp: to test a date filter, two trades on different sides
of the boundary, so the right one drops out.

### Mobile validation

The previous repository rule requiring Android emulator / real-device validation for every UI
change is suspended for this private fork while mobile is outside active scope.

Do not run mobile E2E merely to satisfy a generic completion checklist. Run it only when:

- the issue explicitly includes iOS and/or Android;
- the user explicitly asks for mobile verification; or
- the change is specifically in `mobile/` and requires validation to answer the task correctly.

When mobile validation is required, use the platform-specific instructions in `CLAUDE.md` and
existing mobile tooling.

### Reporting

State plainly what was exercised and what remains unverified. Do not imply that iOS or Android
was tested when it was not. For Web-first issues, "mobile not validated; outside current fork
scope" is an acceptable explicit status.
