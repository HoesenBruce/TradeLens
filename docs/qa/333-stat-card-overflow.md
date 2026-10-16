# Statistic card overflow audit (#333)

## Dataset and method

All records and images used here are synthetic. The user's images and financial records were not uploaded, copied into fixtures, or attached to GitHub.

The disposable API was built from this working tree, with a separate SQLite database on port 8094. The Web app on 5176 used that API. The seed creates JPY, USD and JPY prop accounts, 16 closed trades per account, a breakeven trade, a multi-year holding period, long symbols/mistake names, plans and excursions. The API produces nonempty revenge, overconfidence, loss-aversion, R, compliance and Monte Carlo reports. Goal and prop rules are configured. Native route checks select the JPY account, avoiding dependence on external FX services. Currency conversion in the component boundary fixture uses an explicit synthetic rate; no claim about market exchange rates is made.

Two forms of browser evidence complement each other:

- Codex built-in browser: rendered Home, Reports, recap/share preview and calculator inspections, including the reported paired day cards and actual API selection. Important screenshots remain local.
- Playwright: repeatable geometry and behavior assertions over real components and actual routes. The fixture is a development-only HTML entry; it is not included in the product build. It fetches baseline data from the disposable API. Direct fixture controls supply additional finite amount/count/quantity boundary cases. Goal editor callbacks in that fixture are in-memory component callbacks. The built-in browser also exercised cancel/reopen against the actual Home goal editor and confirmed the API-backed amount was unchanged.

## Inventory and verdicts

`M` means the five supported locales (`en`, `zh-CN`, `zh-HK`, `ja`, `ko`) at 320, 390, 639, 640, 1023, 1024, 1279, 1280 and 1440px. `R` means real route checks in all five locales at 320, 390, 640, 1024, 1280 and 1440px. The tests measure available card tracks and rendered element bounds, rather than assuming a viewport implies a particular card width.

| Surface / callers | Verdict | Evidence and treatment |
| --- | --- | --- |
| Every Home insight: expectancy, best/worst trade, drawdown, PF, win rate + W/L, average hold + count, streaks, winning/losing hold, best/worst day + date, top symbol and main mistake | Changed | M, real API values, long symbol/tag and long duration. Typed money companions use the existing compact formatter; other values wrap without changing their units. Cards grow with their contents. |
| Performance / StatBar / explanation and filter controls | Changed; regression preserved | M, native filter on/off, navigation re-entry and keyboard explanation open/Escape. Full → compact → full and repeated resize assertions. Japanese long metadata reproducer is included in M. |
| Annual goal hero/compact, daily loss and prop program | Changed where needed | M, configured real API rules/prop status. Positive/zero/loss/near-cap/breached/over-target/extreme amounts. Goal modal cancel, save/read-back, clear and restore. Existing compact goal formatting remains intentional. |
| Shared StatCard / StatCell | Changed | M, including long exact counts/qualifiers and symbols. Monetary callers pass explicit full/compact forms; arbitrary strings are never parsed as money. Value/hint truncation was replaced with wrapping. |
| Shared callers: Reports risk, Monte Carlo, rule compliance and all three behavior cards | Changed | M with nonempty API reports, including actual flagged events. Native Reports controls and re-entry/reset. Report formatting explicitly subscribes to the active Lingui locale; an English → Chinese → English test checks the amount strings, not just labels. Trade action rows keep one activation target and full permitted amounts in their accessible names. |
| Shared caller: Reports R performance | Renderer unchanged; shared layout changed | M with populated API R summary. R multiples remain R multiples, counts remain exact. |
| Shared caller: News performance counters | Renderer unchanged; shared layout changed | R for the real report with zero counters; M stresses long count/hint values through the same StatCard. No prediction/accounting semantics changed. |
| Reports overview, context and day strip | Changed | M + R. Actual net/gross, absolute/percentage and mean/median controls, URL read-back after reload, card hide/show, reorder, re-entry, reset and day drawer. Local day-strip scrolling is preserved. |
| Additional Reports amount renderers: period returns, execution grade, hourly list, heatmap detail | Changed | M includes period returns/grade/hourly; heatmap detail popup has a five-locale narrow-width test. Local period-return units remain separate from the global unit toggle. Chart data and aggregation are unchanged. |
| Reports bounded execution score and chart labels; session/table cells | Verified unchanged / not applicable to this renderer change | Existing score scale and compact chart labels are retained. Actual Reports route geometry is checked in R. Deliberately scrolling tables, heatmap grids and chart rendering are outside this statistic-value change. |
| Account balance/equity/realized P&L and Day review stats | Changed | R with JPY amounts and associated help/return percentages. |
| Trade summary, plan/risk-reward, execution cards, peek drawer and trade list cards | Changed | M + R; real peek drawer at 320/390/640/1024px, full-page activation, edit drawer open/cancel. Long identity/tag text, large entry/exit/notional/fees, P&L and R/help qualifiers. Exact quantities are retained. |
| Per-trade, after-save and batch result previews | Changed | M, with large positive/negative/zero values and exact long quantities/counts. The fixture changes values back to ordinary values to catch stale fit state. |
| Day-trades drawer summary | Changed | Native Reports day-card activation, contained full/compact amount and close/re-entry path. |
| Year recap and public share pages | Changed | R, public links with amounts allowed and disallowed. Recap inline monetary translation slots are explicitly tested so fitting cannot remove their values. Long symbols/mistakes wrap. |
| Fixed 1200×630 trade/recap SVG and PNG exports | Changed | M measures every SVG text box. Native recap export produces a PNG which was inspected visually. Existing chip sizing is retained for ordinary labels. Full money is kept when it fits, then existing compact money; native SVG fitting keeps text within its assigned box with a 16px floor. Extremely long names may end in an ellipsis; SVG titles retain the full permitted text. |
| Header aggregate/balance and auxiliary calculator result cards | Changed | Native header on R and original-currency API selection. Calculator ticket, Position size and Kelly dialogs driven at 320/390/wide widths with large values; quantity precision is unchanged. |
| Other non-statistic layout exposed by the same long-name cases | Minimal containment changes | Trade journal tags and rule-note text wrap. Japanese trade-page action buttons wrap. The existing chart interval toolbar uses a local scroll container at narrow widths; chart rendering/replay calculations are unchanged. |

## Boundary and interaction coverage

- Full, existing compact and readable compact-size fallback. A large-card font can reduce to 18px; text already smaller than 18px is not enlarged or reduced. Extreme finite values can use a final ellipsis with complete permitted value disclosure.
- 0, ±10,000, 999,949/999,950, 1,000,000, multi-million JPY, USD, and 1e25, then an ordinary value. Generic count/quantity cases are exact; unavailable `—` values are retained.
- Hover, focus, Enter/Space/Escape, click and an actual touch-enabled browser context. Popup containment, no nested value controls in navigation targets, and privacy on/off while open. Measurement attributes, accessible labels and popup content are checked for leakage.
- Five-locale public pages with `show_amounts=false` contain no monetary value components. Public permissions are enforced by the real API, not simulated with local masking.
- Intentional horizontal scrolling remains local. Geometry assertions ignore hidden measurement copies, local scroll content and the day strip's 4px focus padding, while checking page-level overflow and overlapping sibling cards.

## Reproduce

Use a disposable database, not an existing account database. Public sharing is enabled only in this local test server; market/calendar calls are disabled.

```sh
# api/
TM_HTTP_PORT=8094 TM_DB_PATH=/tmp/stat-card-qa.db \
TM_CORS_ORIGINS=http://localhost:5176 TM_BACKUP_ENABLED=false \
TM_SHARE_LINKS_ENABLED=true TM_MARKET_DATA_ENABLED=false \
TM_ECON_CALENDAR_ENABLED=false TM_JWT_SECRET=$(openssl rand -hex 32) \
go run ./cmd/server

# web/
VITE_API=http://localhost:8094/api/v1 pnpm dev --port 5176
E2E_BASE_URL=http://localhost:5176 E2E_API_URL=http://localhost:8094/api/v1 \
pnpm exec playwright test e2e/stat-card-overflow.spec.ts
```

The suite seeds its synthetic account on first run. Start from a fresh database when changing the seed. Screenshots and the generated PNG are in Playwright's local `test-results/` directory.

## Validation and limits

- TypeScript product check and the separate fixture project check.
- Full Web unit suite: 203 files / 1,262 tests passed.
- Web check: zero errors; existing repository warnings remain.
- Production Web build passed; existing bundle-size/precache warnings remain.
- Browser regression suite: seven acceptance cases (final run recorded in the PR).
- No API contract, accounting, FX/denominator formula, broker behavior or historical calculation changes.
- Mobile not validated; outside the current Web-first scope. No deployment or merge performed.
