# Issue 271 — market timezone for duration and Home

Selective upstream #326 backport; only the market-date portions of #333 are
adapted. No upstream theme/layout changes. Duration uses the requested IANA zone;
absent `tz` retains New York duration classification and legacy UTC daily buckets.
US session classification remains Eastern. Existing account scope, date basis,
backtest-account exclusions and public-share timezone propagation are preserved.

Home's mini-calendar month/year, today marker and today's P&L use market time.
A timer refreshes the day while the page stays open; focus also refreshes it.
Setting the same day causes no React state update.

## Validation, 2026-10-09

- Complete Go suite passed, including Tokyo month rollover, NY DST in both
  directions, 600-second boundary, absent/invalid timezone, existing US sessions,
  multi-account duration filters, public-share account/date scope and default
  backtest exclusion.
- Web suite: 184 files, 1047 tests passed. Static check: 0 errors.
- Fake-clock hook regression holds Home open across Tokyo Jan 31→Feb 1 midnight;
  market switch back to NY restores Jan 31 and switching back restores Feb 1.
  NY spring DST also remains on the correct day. No host clock was changed.
- Built-in browser against a working-tree API and disposable SQLite DB,
  synthetic TZ271 fills 14:55Z→15:05Z Oct 8 (23:55→00:05 JST) and DAY271 fills
  01:00Z→01:10Z (600 seconds).
- NY Day: two trades. Tokyo Day: only DAY271; Tokyo Swing: only TZ271.
  Swing→Day and All reset work. Reload keeps the selected Swing state.
  Switching back to NY makes Swing empty (0 trades); All restores 3 trades.
- General settings read-back shows Tokyo after re-entry. Display zone remains
  New York, independently of the market zone.
- Home mini-calendar today: NY Oct 8 vs Tokyo Oct 9 at the same real instant;
  both directions exercised and rendered screenshots inspected.

Evidence: `tokyo-swing.png`, `new-york-home.png`, `tokyo-home.png`, `new-york-empty.png`.
Midnight/month rollover uses the deterministic hook test rather than waiting for
real midnight. Mobile not validated; outside current fork scope. No data rewrite,
accounting changes, release or deployment.
