# Cash / Margin badges (#344)

Trade list and detail responses now include the read-only `position_type` field:
`cash`, `margin`, or `unknown`. The resolver reuses the execution details already
loaded for holding duration; list requests add no per-trade database/API calls.

Only consistent explicit `sbi:cash`, `sbi:margin-long`, or `sbi:margin-short` lots
establish a known type. Conflicting position metadata, mixed lots, missing fills,
malformed details, and unsupported/manual/non-SBI evidence return `unknown`.
Account labels/capabilities and long/short direction never establish a type.
Clients connected to an older API that omits the field render no badge.

One localized Pill is shared by the dashboard table, narrow list rows, trade
log (shared components), detail drawer title, and full detail summary. Cash and
Margin labels are translated across en, zh-CN, zh-HK, ja, and ko; Chinese displays
现物 / 信用. Existing long/short and conversion event labels remain separate.

No persisted trading data, grouping, quantities, cost basis, P&L, fee allocation,
execution identity, or review associations change. No migration or historical
regroup is required. Unsupported sources display Type unknown rather than a
broker or position type inferred from account settings.

## Automated validation

- `go test ./...` and `go vet ./...` passed.
- Resolver tests cover cash, margin long/short, conversion/delivery metadata,
  absent/malformed/conflicting evidence, and non-SBI defaults.
- API tests persist same-symbol executions and verify list/detail parity for
  open/closed cash, margin long/short, manual, and non-SBI trades; existing money
  and quantity fields are read back consistently.
- Web: 204 test files / 1,278 tests passed, including all five badge locales,
  unknown/older-server behavior, shared table presentation, and drawer identity.
- `vp check`: zero errors (348 existing warnings); TypeScript and production
  build passed. Only the three new entries changed in each compiled catalog.

## Real API Web acceptance (2026-10-17)

Used the Codex built-in browser with this worktree's Web app and a separate
SQLite API on ports 5184/8094. All accounts, executions, and screenshots are
synthetic. The instance was seeded through the API; conversion fixture metadata
was completed only in the disposable QA database, without touching user data.

Exercised and visually inspected:

- Same account and symbol with simultaneous cash, margin-long, and margin-short:
  distinct groups with Cash/Margin badges and unchanged direction indicators.
- Recent Trades to cash drawer to full page, followed by reload/read-back.
- Close/reopen and switching cash -> margin-long -> cash -> margin-short;
  headers reflect the currently opened trade. Escape/close returns to the list.
- Full margin-short detail and reload, a completed margin trade, unknown legacy
  and manual records, and conflicting explicit metadata (Type unknown).
- Margin-to-cash conversion: separate cash/margin groups and the existing
  localized Margin -> Cash execution label on full detail.
- Desktop 1440x1000 and narrow Web 390x844: list badges, drawer title, and full
  detail identity remain readable and wrap with existing layouts.

Badges are read-only; they add no filter, toggle, save/cancel, or reset controls.
No corresponding state-transition tests were invented.

Evidence: [desktop recent trades](recent-desktop.png),
[narrow detail drawer](drawer-narrow.png),
[narrow full detail](cash-detail-narrow.png).

Mobile native apps were not validated; outside current fork scope. No merge,
release, deployment, or real-account historical repair was performed.
