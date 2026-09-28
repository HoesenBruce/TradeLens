# Issue 183 — funding, goals and mixed account selection

## Contract and scope

Header balance/funding return and Reports percentage denominator reuse the normalized Equity cash envelope plus normalized Summary/trade rows. Each cash flow is converted before aggregation; starting_balance is not added twice. Ratios use numerator and denominator in the same API currency, and are invariant under display FX. Empty/unfunded or FX-unavailable scopes hide percentage controls and balance/return values.

Annual goals now persist their source currency (SQLite and PostgreSQL migration 52). GET with target_currency reuses analyticsRate and returns actual currency, latest policy, source_currency and rates. Native reads retain stored amounts. Legacy goals keep their amounts and an empty currency; normalized reads fail with currency_required until the user explicitly confirms and saves an amount/currency. Older clients may still write currency-less goals; they cannot be silently normalized.

Explicit mixed subsets are enabled. Settings annual goals remain user-level/All Accounts; Home/Reports progress respects their current account scope. Currency-less mixed risk thresholds remain unavailable: mixed Compliance/Execution Score reject the scope, and the Home daily-loss card is hidden rather than guessing a currency. Prop status retains its existing single-account boundary.

## Local checks

- Go full suite, vet and build passed; annual-goal API tests cover JPY identity and JPY→USD at fixed 150, unavailable FX, native read preservation, legacy repair boundary, invalid currency/target, clear and empty goal metadata. Migration preserves legacy amounts; dirty migration recovery passes.
- Web full suite: 180 files / 1,038 tests passed. Final focused hook refresh tests: 2 files / 16 tests passed; failed refresh cannot expose cached monetary data.
- Web check: zero errors, 74 existing warnings. Production build passed.
- Existing historical account-value tests passed; no accounting, replay, cost-basis or historical-value code edited.

## Built-in browser acceptance

Current-tree API on 8097, Web on 5186, throwaway SQLite fixture with JPY/ USD / empty accounts and ten closed trades. Live latest Yahoo FX; displayed amounts vary with quote refresh. No production data used.

- All Accounts JPY and USD: cash flows, P&L, balance and funding return normalize consistently (about USD 21.27 P&L, USD 184.79 balance, 13.0%; JPY about 3,347 / 29,085). Explicit JPY+USD subset is selectable, checkmarks read back, deselection produces a single account and reset restores All Accounts.
- Single JPY → USD display: P&L about USD 1.27 and balance 64.79; funding return remains 2%. Reports overall/day return remains 2%, period monthly return 2%; both percent→amount and amount→percent exercised.
- Empty account: zero amounts, no funding percentage or Reports percentage control. Re-entry and reset exercised.
- Annual goal: save USD 1000, read back; edit/cancel/reopen restores saved value. Switch to JPY converts target (~157k), edit/save JPY target tested. Settings Rules edits an explicitly labelled USD target and native API read confirms USD storage.
- Goal FX failure: ZZZ goal renders the real API failure and no progress. Legacy currency-less 500 exposes its retained amount in the error; save explicit USD 500 repairs it. Clear returns unset; setting USD 1000 again restores progress.
- Funding FX failure: disposable ZZZ account cash causes Equity failure; balance is dash and funding/percentage controls disappear, while independently valid P&L remains available. Removing the disposable account and reloading restores funding.
- Settings account cards: each account requests Trades/Equity in its own base currency; JPY balance 10,200/P&L 200 and USD balance 120/P&L 20 remain correct under portfolio JPY display. Goal FX failure repair editor explicitly labels JPY; save 150,000 JPY succeeds.
- 390px: All Accounts goal/progress and editor rendered and visually inspected; cancel restores the page. Screenshots in this directory.
- Home goal and account state survive re-entry; Calendar Month→Year→Month exercised again after final changes.

## Remaining boundaries

Latest conversion is not trade-date FX. No mobile E2E (outside fork scope), no live PostgreSQL browser run. RiskRules monetary currency migration is a separate next step; this work does not invent it. Legacy goal currency must be confirmed by the user. Native broker accounting/detail and historical account-value semantics are preserved.
