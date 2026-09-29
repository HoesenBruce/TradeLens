# Fork Development Policy

## Purpose

This document defines the development scope and compatibility policy for TradeLens, the private fork of [TraderMemos](https://github.com/sinhong2011/TraderMemos),
`HoesenBruce/TraderMemos-Private`.

The goal is to support fork-specific trading workflows while keeping future upstream merges and
possible mobile reactivation reasonably low-risk.

## Current platform scope

The fork currently follows a **Web-first, mobile-preserving** strategy.

### Active delivery target

The Web application is the primary product surface for new fork-specific features.

A Web-focused feature can be considered complete when its API/domain behavior and Web behavior
meet the issue acceptance criteria. iOS and Android implementation, simulator/device testing, and
mobile E2E validation are not required unless the issue explicitly includes mobile support.

### Mobile status

The existing Expo mobile client remains part of the repository, but it is currently in
**maintenance-compatible, not actively validated** status.

This means:

- do not remove the mobile application merely because it is not currently used;
- do not intentionally break mobile contracts when a reasonable shared implementation is
  available;
- do not require mobile-specific UI work for ordinary Web-focused issues;
- do not require Android emulator, Android real-device, iOS simulator, or iOS real-device testing
  for Web-focused feature completion;
- preserving shared contracts does not imply that the mobile client is verified;
- document known mobile incompatibilities instead of silently ignoring them;
- avoid unnecessary platform-specific assumptions in shared API, domain, persistence, market-data,
  importer, analytics, validation, and service layers;
- keep Web-specific implementation details isolated from shared business behavior where practical.

## Future mobile reactivation

If iOS or Android support becomes a product goal again, use the existing mobile client as the
starting point.

The default reactivation process should be:

1. restore the selected mobile platform(s) to the active acceptance scope;
2. run compatibility, build, and end-to-end testing against the current API;
3. identify regressions and unsupported flows accumulated during the Web-first period;
4. fix shared-contract incompatibilities first;
5. add platform-specific adaptations where required;
6. restore automated and manual validation for the reactivated platform(s);
7. update issue templates/checklists so new work includes the reactivated platform going forward.

Do **not** assume the mobile client must be rebuilt from scratch. A rewrite should require a
separate architectural justification based on the actual condition of the existing client.

## Shared-code rule

Fork-specific development should prefer reusable shared behavior when it is reasonable to do so.

Examples include:

- API contracts;
- domain models;
- importer semantics;
- persistence and validation behavior;
- market-data abstractions;
- analytics calculations;
- symbol/instrument metadata;
- service interfaces.

A Web implementation may use browser-specific presentation or interaction code, but shared
business behavior should not become Web-only without a clear reason.

This policy is not a requirement to pre-implement mobile UI. It is a requirement to avoid
unnecessary architectural dead ends that would force a future mobile rewrite.

## Web-first acceptance policy

For ordinary fork issues:

- relevant API tests must pass;
- relevant Web tests must pass;
- changed Web workflows should be exercised end to end;
- loading, empty, error, and important state-transition behavior should be tested where applicable;
- lint and type checks should pass for affected components;
- mobile implementation and testing are optional unless explicitly required by the issue;
- any known mobile regression or incompatibility introduced by the change must be documented;
- completion reports must not imply that iOS or Android were validated when they were not.

`AGENTS.md` contains the operational UI verification rules.

## Upstream compatibility

This is a private fork, but changes should still minimize avoidable conflict with upstream.

Prefer:

- additive changes over destructive rewrites;
- focused modules over broad refactors;
- extension of existing abstractions over parallel duplicate stacks;
- preserving existing public API semantics where possible;
- documenting fork-only behavior clearly;
- isolating provider- or broker-specific logic behind interfaces;
- keeping unrelated formatting or cleanup out of feature changes.

Before finishing a substantial feature, review the final diff for unnecessary merge risk.


## Broker and accounting compatibility

Broker-specific accounting is an **opt-in extension** in this fork. The default behavior for
existing brokers should remain compatible with upstream unless a separate issue explicitly
authorizes a semantic change.

### Default behavior

For IBKR and any broker / transaction combination without a verified fork-specific accounting
rule:

- preserve the current upstream/default trade grouping and P&L semantics;
- preserve existing importer, fee, commission, multiplier, and dedup behavior;
- do not change accounting merely because an account is marked `margin` or has a `margin`
  capability;
- do not use instrument type, exchange country, or market alone to select a different accounting
  method;
- do not opportunistically fix unrelated upstream accounting behavior inside a broker-specific
  feature.

Unknown or unsupported broker/transaction combinations must fall back to the upstream/default
path rather than guessing a specialized rule.

### Broker-specific opt-in

A specialized accounting rule may be introduced only when its source semantics are documented and
covered by tests. Strategy selection should use explicit context such as:

```text
broker/source + execution/position semantic
```

For example:

```text
SBI + cash
→ verified SBI/Japanese cash-equity strategy

SBI + margin_long / margin_short
→ verified SBI margin-settlement strategy

SBI + genbiki position conversion
→ verified SBI conversion strategy
```

Do **not** assume that a rule verified for SBI automatically applies to every Japanese broker.
Add another broker only after its export fields and transaction semantics have been checked.

### Account capabilities are not accounting rules

Account metadata and transaction accounting are separate concerns.

A brokerage account may expose capabilities such as:

```text
cash
margin
options
futures
```

Those capabilities describe what the account can contain. They must not by themselves decide how a
specific execution is costed or matched.

For example, an IBKR margin account and an SBI account with Japanese 信用取引 both have margin
capability, but they must not be assumed to share the same position or settlement semantics.

### Shared-engine changes

When broker-specific support requires touching shared trade/P&L code:

- keep the existing upstream/default path intact whenever practical;
- prefer a small resolver/strategy seam over rewriting the default engine;
- make specialized behavior explicit rather than inferred from generic buy/sell direction;
- add regression tests for representative non-target brokers before calling the change complete;
- preserve current default partial-close semantics unless the issue explicitly changes them;
- do not normalize an inherited frontend/backend P&L difference as incidental work.

A change that intentionally alters IBKR or another existing broker's accounting semantics requires
its own issue, explicit before/after examples, migration/recalculation impact analysis, and
regression coverage.

### Required regression protection

For accounting/importer changes that touch shared code, verify representative existing behavior as
applicable:

- IBKR simple stock round trip;
- IBKR commission handling;
- IBKR option multiplier/contract handling;
- generic importer behavior;
- at least one other affected existing broker preset;
- historical/import dedup behavior when the import path is touched.

The acceptance report should state both the broker-specific behavior added and which non-target
broker paths were verified unchanged.

## Issue scope guidance

An issue should explicitly identify the platform scope when UI work is involved.

Recommended wording for current fork work:

> Platform scope: Web is the active delivery target. iOS and Android are not required acceptance
> targets for this issue. Shared API/domain changes should remain reasonably compatible with
> future mobile reactivation, and known mobile incompatibilities must be documented. Mobile is
> not considered tested unless the issue explicitly requires and records mobile validation.

Do not create Android/iOS implementation or validation sub-issues merely because the upstream
project supports those platforms.

If a mobile-only issue already exists and is no longer part of the current roadmap, close it as
`not planned` rather than deleting it. This preserves context for future reactivation.

## Scope changes

When mobile becomes active again, update this document first or as part of the issue that changes
the platform policy. Acceptance criteria and agent instructions should then be updated together so
that repository documentation does not contradict the active development scope.

### Web i18n convention

- Use the existing Lingui provider (`web/src/i18n/index.tsx`) and `useLocale()`.
  Required locales are `en`, `zh-CN`, and `ja`; preserve upstream `zh-HK` and `ko`.
- Navigation and Settings resources live in `web/src/lib/locale.ts`. Their typed
  `navLabel(locale, "home")` / `settingsLabel(locale, "language")` APIs use semantic
  keys, equivalent to `nav.home` / `settings.language`. Add new labels to English
  first, then the supported translations. Do not use display strings as keys.
- Other UI uses Lingui catalogs in `web/src/i18n/locales/{locale}/messages.po`.
  For new fork strings, use explicit semantic IDs, e.g.
  `t({ id: "trade.import", message: "Import trades" })` from Lingui's React macro
  hook. Run `pnpm i18n:extract`, translate the entries, then `pnpm i18n:compile`
  from `web/`; commit the relevant PO and compiled TS changes. Existing upstream
  display-string IDs can remain until their surface is migrated.
- New user-facing text in localized surfaces must use these APIs, not hard-coded
  JSX. Fallback is selected translation → English → visible key. Typed label keys
  catch typos; Lingui warns about unknown IDs during development.
- The device-local `tm-locale` preference wins at startup. When absent, match
  `navigator.languages`; otherwise use English. Invalid saved values use English.
  Changes apply immediately; this selector has no separate save/reset operation.
- Reuse `fmtDate`, `fmtTime`, `fmtNumber`, `fmtPct`, and `fmtMoney` in
  `web/src/lib/format.ts` with `useIntlLocale()` for reactive formatting. Date/time
  helpers preserve display timezone/clock preferences; money preserves privacy.
  Percentages take ratios (0.58 = 58%). Locale must never change currency codes,
  stored dates, API enum/filter values, or financial calculations.
- Issue #85 covers navigation and General settings as the representative surface.
  Remaining UI/catalog entries may fall back to English; full translation and
  layout QA belong to a later issue once the UI stabilizes. Mobile is preserved
  but localization/device behavior has not been validated.

### SBI margin settlement P&L and split boundaries (#148)

Completed SBI margin trades with broker-reported close results use the sum of
settlement P&L (plus any supported single-opening fallback closes). Settlement
values are already net: fees are not deducted again; gross is net plus the
recorded fees. Ordinary partial closes keep `trade.net_pnl = null`. The existing
conversion reconciliation remains in force. Other broker buckets retain their
normal accounting unless a corporate-action boundary makes the price basis unsafe.

For example, two 7013 buys of 100 at 17,980 and 17,510 followed by two sales of
100 at 2,614.5 and 2,611.5, with SBI settlement results of 11,331 and 11,031,
now produce net JPY 22,362 rather than JPY -3,026,438 (assuming JPY 38 fees).

With market data enabled, regroup checks daily provider split events and the
existing OHLC discontinuity detector (including 7:1). A crossing trade without
complete SBI settlement evidence gets null gross/net P&L and return percentage.
With complete evidence, settlement P&L is retained but return percentage is null:
raw entry notional is not a split-adjusted return basis. `accounting_warning` is
persisted and included in list/detail responses, the detail drawer and full page.
No split normalization of execution quantities or prices is performed.

Market lookup failures abort regroup before trade writes; retry after restoring
the provider. Disabled/unconfigured market data cannot establish split boundaries
and is explicitly labelled as unchecked on multi-day stock streams. Detection is
limited to provider coverage and the existing heuristic, not an authoritative
corporate-action registry. Offline import/legacy normalization callers likewise
need a configured bars getter to perform the check.

Existing trade rows are not rewritten by the schema migration. Rebuild affected
accounts via `POST /api/v1/trades/regroup` with `{"account_id":"..."}` and working
market data. This can change historical P&L/statistics or exclude unsafe results
from P&L aggregates; original imported executions and journal annotations remain.
