# Issue 270 — OCR fill instants

Selective backport of upstream #316/#317, with TradeLens scan-overlay coverage,
per-row timezone provenance, mixed-image ordering, blank unknown times, and
original-instant preservation through an unchanged form (including DST folds).

The source clock is the row label, then screenshot label, then submitted market
`tz`, then America/New_York. Model-appended offsets alone remain untrusted.
Yearless dates use the recent year heuristic; review the inferred year before save.
Date-only, unreadable and nonexistent DST wall times require manual correction.
Only completed orders are extracted; partial/cancelled/working orders are excluded.
Retries: transient 429/500/502/503/504, at most two retries (1s/3s), no timeout retry.

## Local acceptance, 2026-10-09

Built-in Codex browser, working-tree API with a disposable SQLite database,
synthetic local OpenAI-compatible vision provider, existing test PNG. No real
broker image or external vision credentials used.

- Scan JST 2026-10-01 10:30:27 / 10:31:27; New York review/form shows
  2026-09-30 21:30:27 / 21:31:27.
- Cancel review leaves the original form empty; scan again and Fill form works.
- Reopen time picker: September 30, 21:30:27; Cancel preserves that value.
- Save and read back from real API: opened_at 2026-10-01T01:30:27Z,
  closed_at 2026-10-01T01:31:27Z, closed round trip +4 USD.
- Cancelled synthetic order absent from review and persisted trade list.
- Unreadable time shows warning, leaves a blank editable time; Save blocked.
- Clear returns to empty symbol/fill baseline; Cancel returns to the saved list.
- Rendered screenshot inspected: `saved-trade.png`.

Go suite and Web checks/tests recorded in the PR. No schema migration,
historical rewrite, deployment, or broker accounting change. Mobile not validated;
existing mobile clients still need an independent source/display-time acceptance pass.
Live LLM recognition quality is not established by this deterministic provider test.
