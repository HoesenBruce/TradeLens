# Issue 179, part 1: Equity and Daily

Parent #179 is split into independently mergeable tasks, in dependency order:

1. [#181](https://github.com/HoesenBruce/TraderMemos-Private/issues/181): this PR, Equity and Daily currency inputs.
2. [#182](https://github.com/HoesenBruce/TraderMemos-Private/issues/182): grouped analytics and row consumers (Breakdown/Reports/Playbook, trade lists, Calendar row statistics, account contribution, Wrapped).
3. [#183](https://github.com/HoesenBruce/TraderMemos-Private/issues/183): Header/funding/annual goals and explicit mixed portfolio selection, after all consumers support the contract.

## Contract

Summary, Equity and Daily share the normalization extracted from #175 / PR #180.
Each persisted trade's `pnl_currency` supplies the source for net/gross P&L and fees.
Each cash-ledger entry supplies its own `currency`, independently of account metadata.
Conversion happens before Daily date buckets, Equity chronology and monetary drawdown.
`target_currency`, actual `currency`, `fx_policy: latest`, and `fx_rates` use the existing market service/cache.
Invalid/unavailable FX returns `502 fx_unavailable` without points, buckets or cash totals.
Mixed scopes without a target remain rejected. Unknown scopes/currencies fail explicitly.
Latest FX is not historical trade-date FX.

With an explicit target, Daily returns `{currency, target_currency, fx_policy, fx_rates, pnl}`.
Without a target, same-currency Daily retains its legacy date-to-amount map for existing clients (including mobile).
Equity adds metadata and `cash_transactions` normalized to its output currency, preserving existing points/drawdown fields.
Web requests explicit targets for Daily/Equity: mixed = display preference, Auto = USD; single currency = native currency.
Query keys include targets; each rendered monetary surface uses response currency.
Calendar uses Equity's normalized cash for balance/deposit readings, rather than raw account balances or ledger sums.
Unavailable trade-row detail statistics and year trade counts are hidden, rather than presented as zero.

## Verification

- Go full suite, vet and build passed; existing #171 account-value tests passed.
- Currency API checks: JPY+USD to JPY and USD; All Accounts, explicit mixed subset, single account, empty trades, backtest exclusion, invalid target/unknown ID, missing FX, cash currency differing from account currency, and monetary drawdown.
- Existing Summary normalization regressions passed after extracting its shared path.
- Web full suite: 179 files / 1029 tests passed before the final unavailable-row guard; final affected suite: 4 files / 61 tests passed (includes that guard and historical account-value hooks).
- Web check: 0 errors / 74 existing warnings. Production build passed. `git diff --check` passed.
- Built-in browser: real isolated SQLite API 8097 and current-tree Web 5186. Fixture JPY +200 (Sep 7), USD +20 (Sep 8), opening deposits JPY 10000 and USD 100, empty JPY account. Live latest USD/JPY approximately 157.50; JPY total 3350 / USD total 21.27 at display precision.
- Home: rendered Equity axis uses JPY then USD, Daily dates and total follow the target once. Historical Account Value remains separately normalized.
- Calendar: JPY 200 + 3150 vs USD 1.27 + 20; matching week/month/year totals. Month/year both directions, prior empty month and return, empty account and recovery.
- Account transitions: All -> JPY -> All -> USD -> All -> empty -> All. Single scopes exclude the other currency's date. Currency transitions USD/Auto -> JPY -> USD/Auto -> JPY; cancel, re-entry, checked read-back and reload exercised.
- FX failure: disposable ZZZ trade made All Accounts fail closed; Calendar showed an error with no partial P&L; removing the disposable fixture restored results. Mixed trade drawer continues to show its unsupported-path error, not fabricated row amounts.
- 390px rendered Calendar verified; year amounts remain visible and unavailable trade counts show a dash. Viewport override restored.
- Dev server initially retained stale transformed modules after edits; restarted and re-exercised final unavailable-row guard against current code.

Screenshots: [Home](home-jpy.png), [Calendar](calendar-jpy.png), [Reports](reports-jpy.png), [390px year](narrow-year.png), [FX failure](fx-failure.png).

## Remaining scope and boundaries

#182 covers Calendar row-derived fees/PF/expectancy, mixed day/week rows and trade counts. This PR enables the P&L heatmap and normalized balance inputs, not the entire mixed Calendar trade experience.
Breakdown/Playbook, mixed trade lists/account contribution/Wrapped, Header balance/funding return, annual goals and incompatible-currency explicit selection retain their existing restrictions until #182/#183.
No broker accounting, cost basis, position replay or #171 historical reconstruction changes. No stored money or historical statistics recalculation.
Mobile contracts preserved; mobile not validated, outside current fork scope.
Unrelated `api/cmd/diag_account_value/` preserved and excluded.
