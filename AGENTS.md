# Agent rules

## Private-fork platform policy

This repository currently follows a **Web-first, mobile-preserving** development policy.

The Web application is the active product and acceptance target for fork-specific feature work.
The existing Expo mobile application remains in the repository, but iOS and Android are not
currently required delivery targets unless an issue explicitly opts them back in.

For ordinary fork development:

- implement and verify the Web experience when UI work is requested;
- keep shared API, domain, data-model, and service changes reasonably platform-neutral;
- reuse existing shared abstractions instead of introducing Web-only assumptions without need;
- do not remove, rewrite, or intentionally break mobile code merely because mobile is currently
  outside active validation scope;
- mobile-specific implementation is optional unless an issue explicitly requires it;
- Android emulator / real-device and iOS simulator / real-device validation are **not required**
  to complete a Web-focused issue;
- if a change is known to leave mobile unsupported or unverified, document that limitation in
  the issue or PR rather than expanding scope automatically;
- when mobile support is reactivated, treat the existing mobile codebase as the starting point:
  run compatibility testing, fix regressions, and add platform-specific adaptations as needed
  rather than rebuilding the client from scratch by default.

See `docs/FORK_DEVELOPMENT.md` for the canonical scope and upstream-compatibility policy.

## Web UI features ship only after a full end-to-end run

**Any new or changed Web UI must be driven end to end before the work is called done or a PR
is opened. A feature with an unexercised required Web case is unfinished, not "probably fine".**

Static checks and screenshots are useful but do not replace exercising the behavior.

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
