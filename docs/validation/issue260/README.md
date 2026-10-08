# Issue #260 — deployment reference audit

Audited tracked files on 2026-10-08, starting from `c47e647a` (origin/main).
[references.txt](references.txt) lists all 319 matching source occurrences after cleanup,
with Preserve/Fix/Review classifications. Search decoded percent-encoded URLs as well
as literal `sinhong2011`, legacy hosted product URLs, provider deploy/clone/template
URLs and URLs containing `tradermemos`. Binary assets are excluded.

## Decisions

- **Fix:** Railway's upstream template comment now directs users to manual TradeLens
  GitHub import. Removed unverified Railway template URLs from fork/deploy and Web
  README; setup links replace deploy badges. Vercel/Cloudflare/Netlify source targets
  in Web README and localized fork guides use `HoesenBruce/TradeLens`. Cloudflare's
  config comment agrees. Explicit YOURUSER URLs remain examples to customize.
- **Fix:** all four fork guides distinguish `web/` SPA, separate Go API and
  `marketing/` Next.js. Marketing README explains its independent deployment.
  Localized self-hosting warnings now agree with Compose's official GHCR defaults,
  retaining the prohibition on sharing upstream/fork databases.
- **Preserve:** NOTICE/license/authorship, upstream changelog/history/plans,
  sponsorship, upstream demos and inherited documentation links. The Marketing
  docs layout visibly labels inherited content as upstream reference. Root README
  already distinguishes SPA/API deployment, upstream mobile and upstream demo.
  Marketing Home already links to fork guides; no component change was necessary.
- **Review:** inherited security/discussions routing, tm-sync release links,
  `web/src/lib/version.ts` upstream repository constant, mobile ownership/about
  URLs and app identity require a separate ownership/compatibility decision.
  They are not advertised deployment templates and remain unchanged.
- **Preserve compatibility:** no `TM_*`, database filename, Compose volume/service,
  export envelope, Worker identity, API behavior or mobile identity is renamed.

## Validation

- `git diff --check`: passed.
- URL-decoded scan: no actionable provider link targets upstream; no unverified
  Railway template URL remains outside this audit's historical explanation.
- Python `tomllib`: Railway and Wrangler parsed configuration equals HEAD exactly
  (only comments changed); Netlify config parses. NOTICE, LICENSE, Compose,
  Web app-config envelope and mobile/app.json match HEAD byte for byte.
- 61 local/locale Markdown links in changed documentation resolve to files/routes.
- Codex built-in browser, local Next.js preview on port 3100: all four fork-deploy
  pages rendered and were visually inspected; screenshots below. Twelve localized
  deploy/configuration/updating/backup-restore pages opened and their rendered
  content contained official GHCR defaults and no stale upstream image default.
- Clicked the Netlify deployment link: a new provider tab named TradeLens as the
  repository and requested GitHub connection. Did not connect an account or deploy.
- Clicked Railway section anchor, returned with Back, and reopened/reloaded docs;
  scope notes and corrected link targets persisted. No changed form, filter,
  toggle, cancel or reset behavior exists. These documentation pages need no API
  data; no journal API/state was used for acceptance.
- Initial sandbox preview hit EMFILE; restarting outside the sandbox with Webpack
  and polling allowed page compilation and browser checks to complete.

## Provider verification limits

Read-only entry checks: [Railway new project](https://railway.com/new),
[Vercel clone](https://vercel.com/new/clone),
[Cloudflare deployment](https://deploy.workers.cloudflare.com/) (redirects to dashboard),
and [Netlify deployment](https://app.netlify.com/start/deploy).
These establish entry-page reachability only. Private repository authorization,
clone/build success, paid resources, volume attachment, CORS and complete deployed
stack behavior require the owner's account and remain unverified. No working
Railway template is asserted. No deployment, release or automation was changed.

No Go/Web application tests or full production build were run for documentation
and comment-only changes. Mobile not validated; outside current fork scope.

## Screenshots

- [English](en.jpg)
- [Japanese](ja.jpg)
- [Simplified Chinese](zh-Hans.jpg)
- [Traditional Chinese](zh-Hant.jpg)
