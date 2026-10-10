# Issue #289 local Web acceptance

2026-10-09, Codex built-in browser, localhost:5193, isolated SQLite API on :8093.
No production data was used. The tested working tree contained this fix on top of
v0.3.0 (`60b8c9e4223c0890f98a4423cce0920eda4da398`); the displayed SHA is the base
checkout/build metadata, not a claim that these uncommitted changes were deployed.

- `289-live-release.png`: real GitHub TradeLens stable feed, installed Web/API
  v0.3.0, latest v0.3.0, current status, TradeLens release link. Upstream attribution
  and product links were also checked in the rendered About header.
- `289-newer.png`: controlled v0.4.0 response, Web/API behind and View release.
- `289-unavailable.png`: controlled prerelease response, neutral unavailable state,
  previous update/release action cleared. A 503 response was exercised too.

Controlled responses were supplied by a temporary Vite network fixture, removed
before committing; the API and authentication remained real. Exercised current →
newer → failure → retry/newer → prerelease → no release (404) → current. Leaving
About and reopening preserved the applied release state. No cancel/reset flow
exists for this check; retry and return to current are its reverse transitions.

Web unit suite: 185 files / 1,067 tests passed. Targeted release/update/About suite:
46 tests passed across the targeted runs. Web check and production build passed (existing lint warnings and
jsdom scrollTo warnings remain).

Public baseline `/healthz` reports v0.3.0 without a commit field. The public demo
has not deployed this fix; version/SHA and update-source regression must be checked
after the next authorized release/deploy. Mobile validation is outside this issue.

Local demo Docker image built successfully. An isolated container smoke check
confirmed `/healthz.version` v0.3.0 and the supplied full SHA in API health and
Web assets. Render itself still requires post-release verification.
