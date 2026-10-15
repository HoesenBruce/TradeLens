# Issue #317 — closed / remaining quantity progress

Validated on 2026-10-15 against a disposable real SQLite API with synthetic data,
after #316 merged as #321 (8485986).

## Source of truth and scope

The existing `trade.quantities` read model from #308/#310 is authoritative:
`entry_qty`, `exit_qty`, `open_qty`, `remaining_pct`, `lifecycle`, and
`unknown_reason`. It comes from the reconstructed inventory for the selected
trade segment, including reversal/conversion boundaries. The renderer does not
sum gross fills or infer current exposure from cash/margin account labels.

No importer, accounting, matching, fee, P&L, persistence or API contract changes
are included. Existing inventory rounding remains in force: the default engine
stores remaining quantity to two decimals. The large fixture therefore reads
1,234,567,890,123.125 opened and 123.13 remaining; the UI displays those API
values, including the derived exit quantity, without replacing them with gross
execution quantities. Supported fractional entry units retain their decimals.

## Presentation

- List rows, compact Web cards, the peek drawer and full details share the same
  closed/remaining progress semantics. 500 opened / 400 closed / 100 remaining
  shows 80% closed and 20% remaining.
- The detail quantity section occupies the full available width and stacks
  below 40rem of container width. Values wrap and have no ellipsis.
- Fully closed, partially closed and open positions have explicit text labels;
  color is supplementary. Unknown/missing history stays unknown, with its
  explanation and no invented zero-percent bar.
- Progress is bounded to 0–100% for the selected segment. Values close to the
  boundaries use `<0.1%` / `>99.9%` rather than falsely displaying 0% or 100%.
- Bars expose accessible names, bounds, numeric values and full textual
  descriptions. Quantity labels and values remain visible; no hover is needed.
  The compact card retains keyboard activation and Escape closes the drawer.
- The table column remains sorted by remaining quantity, with an explicit hint.
  Virtualized table rows measure their rendered heights so multiline progress
  rows scroll correctly.

## Verification

- Web check: 0 errors (existing lint warnings remain).
- All 200 Web test files / 1,246 tests pass; final affected component tests
  pass after row measurement changes; TypeScript and production build pass.
- API quantity persistence/read-back and holding integration tests pass;
  importer and trade accounting suites pass. Backend code is unchanged from
  the fully validated #316 base.
- Joint Playwright run passes both `trade-holding.spec.ts` and
  `trade-quantities.spec.ts` against the actual API on :8092 and Web on :5175.
  Coverage includes Manual, IBKR, generic CSV and SBI inputs; partial, full,
  never-closed, short, pyramid, reversal, conversion and unknown inventory;
  list sorting in both directions, filter clear/reset, close/reopen/reload,
  full details, English/Chinese/Japanese, full numeric values at 1440px/390px,
  keyboard entry to the compact drawer, zero horizontal page overflow, and a
  30-row variable-height table scrolled to the end and back.
- API quantities, fees and monetary fields remain unchanged after all viewing.
  The #316 duration E2E also passes with the new layout.
- Screenshots below were visually inspected. The built-in browser was
  unavailable, so acceptance used Playwright Chromium. External market-data
  feeds were disabled in the disposable API.

## Evidence

- [Desktop list](list.png)
- [500 / 400 / 100 detail](partial-detail.png)
- [Fully closed detail](full-detail.png)
- [Long quantities in peek drawer](long-peek.png)
- [Compact long-quantity list](long-compact-list.png)
- [Compact long-quantity drawer](long-compact-peek.png)
- [Chinese compact detail](long-compact-detail-zh-CN.png)
- [Japanese compact detail](long-compact-detail-ja.png)
- [Measured virtualized rows](virtualized-quantity-rows.png)

Native iOS/Android were not validated; outside current fork scope.
