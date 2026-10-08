# Screenshot provenance and capture workflow

## Issue #254 showcase captures — 2026-10-08

**Status: unfinished.** Eight actual production Web captures are available in `showcase/`;
SBI browser upload/preview/confirm/result remains blocked. API import and the broker-guide
image do not replace end-to-end browser upload acceptance. Under `AGENTS.md`, do not open
a PR until this required Web case is completed.

Use [showcase setup and audit](../showcase-demo.md) from #253, source commit `eef3752d`.
All inputs are synthetic literals, with no RNG: 12 SBI rows, 14 executions, seven closed
groups, JPY 25,000 P&L, 950,000 contributed capital and 975,000 balance/account value.
The local Generic Bars fixture covers six September 1–8 JP sessions. Prediction counts
are six horizon samples: four pending, two unavailable, zero validated/incomplete.
Server-owned save timestamps cannot be backdated through supported APIs. No scored
outcome or live AI analysis is fabricated.

### Reproduce the capture setup

Run the local market fixture and a fresh SQLite API per the showcase guide. Set
`TM_CORS_ORIGINS` to the preview origin. This run used API port 8098, fixture 18964
and Web 4174. `scripts/test_showcase.py <api-base>` creates the disposable user and
checks the seed; its duplicate-import test leaves a second import-history batch with
zero new executions, visible in the cash-flow capture.

```sh
cd web
VITE_API=http://127.0.0.1:8098/api/v1 pnpm build
pnpm exec tsc -b --noEmit
pnpm run preview --host 127.0.0.1 --port 4174
```

Log in with the fictional credentials from the showcase guide. Use English, Dark, JPY,
the fictional account, Tokyo display/market timezone and a **1440×1000** viewport.
Captures use the Codex built-in browser, without chrome. JPEG viewport pixels are
unedited; dimensions, metadata absence and hashes are checked in the manifest.
OS-native date controls can retain the browser locale despite English app labels.

| File | State |
| --- | --- |
| `portfolio.jpg` | Home; Equity and Historical account value, ALL ranges |
| `cash-flow.jpg` | Settings → Accounts; signed deposit and withdrawal |
| `trades.jpg` | Seven closed groups; Symbol, Status, Direction, Qty, Entry, Exit, P&L, P&L %, Created at, Close date |
| `plan-review.jpg` | 6501 full trade; long plan target 1,200, actual exit 1,100; 1D fixture chart |
| `sbi-guide.jpg` | Import → Find my broker → SBI Securities; existing fictional account |
| `news-thesis.jpg` | 285A fictional thesis, asset and manual Bearish 60% prediction |
| `prediction-create.jpg` | Manual editor; synthetic Bullish 70%, 1D/5D input |
| `prediction-performance.jpg` | Six samples; four pending, two unavailable |

The editor capture was followed by Save, reload read-back, edit/Cancel/re-entry and
public API cleanup of that extra record. Source=AI excluded all samples; reload retained
that filter and Reset restored six samples. Portfolio/trade/ledger values were checked
against the running API, not intercepted or cached responses. All eight images were
visually inspected: no real records, email, user ID, private URL or credential appears.

### Remaining acceptance and limitations

Built-in file-chooser events timed out. Safari was being used for another task and was
left alone. A separate Chrome tab reached the native picker, then macOS ScreenCaptureKit
failed to capture UI state. No external-browser image is included. Resume with a working
chooser and fresh disposable API; capture actual SBI preview/confirm/result and duplicate
import before marking #254 complete.

No Web code changed. Mobile, live SBI/AI/market data, Postgres and remote CI were not
validated. List/detail prediction cells remain Pending placeholders; the performance
report and validation API supply actual outcome evidence. The fixture has no intraday
coverage for MAE/MFE or intraday replay.

### Reference audit

README uses five new showcase captures. Existing root PNGs and all five marketing copies
are retained with their #211 hashes under `retained_legacy` in the manifest. Marketing
continues to label its generated USD demo and metrics; its information architecture and
branding are separate scope. Verified all 22 new/legacy image hashes, four fixture hashes, 26 documentation image links
and marketing screenshot references. No referenced image was removed. Root/marketing PNGs are
fork-generated captures, not unlabeled upstream images. Historical validation folders
below retain their original provenance and were not recaptured in this run.

## Retained legacy USD captures — issue #211, 2026-09-30

All nine root Web PNGs in this directory and five Web PNGs in
`marketing/public/screenshots/` were newly captured from a production Web build
in the Codex built-in browser. Duplicate README/marketing names contain the
same newly captured pixels. No old screenshot was redacted, cropped or reused.
PNG conversion retains pixels only; EXIF/text metadata is absent.

Inputs: #210 `scripts/seed-demo.py`, RNG seed **210**, fixed end date
**2026-09-29 UTC**, a fresh throwaway SQLite API database, and an invented
`demo@example.com` user with a `Generated Demo` USD account. The generator
POSTed 186 round trips; existing grouping produces **183 closed trades**, net
**$8,398.72**, and **$33,398.72** balance from a generated $25,000 deposit.
The marketing statistics and decorative trade rows were synchronized to these
generated results; the unconfigured goal-pace metric was removed. All four
marketing locales label screenshots/statistics as generated demo data.
These are illustrations, not actual trading performance. No live broker,
owner account, exported file or production database was used.

Reproduce on a new throwaway API using the working tree:

```sh
# From api/: configure a fresh /tmp DB, local HTTP port and JWT secret;
# set TM_CORS_ORIGINS to the Web origin if the API is on another port.
go run ./cmd/server
# POST /api/v1/setup with demo@example.com, a test password and a USD account
# named Generated Demo, starting_balance 0. Then from the repository root:
python3 scripts/seed-demo.py --api http://localhost:<port>/api/v1 \
  --email demo@example.com --password '<test-password>' \
  --seed 210 --end-date 2026-09-29
cd web
pnpm build
pnpm run preview
```

Configure the browser's API server to that throwaway origin and select English.
Capture Home in Dark and Light, Reports → Overview (All chart range), Calendar
→ September 2026 Month, Playbook, Trades, and the empty sign-in page after logout.
Trade-list columns used: Symbol, Status, Direction, Market, Qty, Entry, Exit,
P&L and P&L %. Captures use the browser's default 1002×871 viewport. Clear the
custom API setting before capturing the login page. Do not capture Settings
with server URLs or user/account identifiers visible.

Visual review checked all seven unique captures: current TradeLens icon/name,
English labels, generated prices/P&L, no names/emails/account IDs/server URLs
or notifications. Browser chrome is not included. Login has the language
selector and empty credentials. Recorded images/hashes are in
[screenshot-manifest.json](screenshot-manifest.json).

## Retained validation images

- `news-predictions/` and `news-analysis/`: disposable API database and API/UI
  fixtures described in [news-predictions](../features/news-predictions.md).
  Source helpers are `web/e2e/news-{predictions,list,detail,analysis}.spec.ts`;
  AI responses are local fixtures, not private provider/account data.
- `news-performance/`: procedural prediction/bar cases in
  `web/e2e/news-performance.spec.ts`, described in
  [prediction-validation](../features/prediction-validation.md).
- `docs/validation/issue{175,179,182,183}/`: explicit throwaway JPY/USD seeded
  cases documented in each directory's README and recorded as safe by #207.
- `docs/validation/issue191/`: About/attribution screenshots from the documented
  throwaway API/QA user and account (see its README); no real-account input.
- `docs/validation/issue200/`: empty-login branding captures; no trading records.
- `web/e2e/fixtures/fill-confirm.png`: 1×1 upload placeholder, no financial
  image content; OCR data is generated by the route fixture in `ocr.spec.ts`.

Application icons/brand art are non-financial assets. Historical financial QA
images in `docs/qa/issue-109/` and `docs/validation/issue103/` were removed;
text validation records remain. Old mobile README/marketing captures and all
store artboards/raw captures/video were removed with the owner's approval.
Native recapture is required before mobile/store publication; scripts remain,
and this PR makes no mobile validation claim. Historical Git blobs still need
#207's separate history-remediation and final publication audit.

## Validation

- Web production build and built-in-browser login, real API read-back, Home,
  Reports, Trades, Calendar, Playbook, dark/light theme round trip and logout
  exercised. Seven unique images visually inspected.
- Marketing lint, TypeScript/MDX type checks and production build passed.
  Built-in browser verified the generated metrics/disclaimer, replacement
  homepage images in both themes, and the English mobile guide with its removed
  image block. Four localized mobile guides compile without stale image links.
- All fourteen replacement PNG files are metadata-free, their hashes match the
  manifest, and repository searches find no dangling removed image links.
- Marketing retains its existing upstream site identity and mobile download
  links; changing that site's branding/distribution scope is separate work.
- No native mobile runtime was available; no native validation or new store
  imagery is claimed. The owner chose deletion rather than retaining unproved
  mobile captures.
