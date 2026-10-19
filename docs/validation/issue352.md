# Issue #352 — SBI import guidance

## Scope and implementation audit (2026-10-19)

Based on current `origin/main` at `2b13551a68947df40746a2b5dbebd627d8eadf53`.

| Related work | State checked | Current behavior used by the guide |
| --- | --- | --- |
| #201 | Closed; basic guidance present on main | Execution and cash CSV sources, uploaded separately; Japanese broker menu labels |
| #312 / PR #313 | Merged, `5a10dfe3` | Retained CSV execution/journal sources can be reprocessed from import history; legacy, cash-only and enrichment imports cannot |
| #162 / PR #164 | Merged | Realized-P&L CSV supplements existing margin closes |
| #338 / PR #341 | Merged, `b14bbff1` | Cash-sale and delivery rows use the same enrichment preview/apply flow |
| #350 / PR #351 | Merged, `460585c4` | Close matching does not require proof of opening-lot identity; unresolved matches are not guessed |

Read `ImportView.tsx`, settings import history, `import_handlers.go`, `import_sources.go`,
`sbi_margin_pnl.go`, SBI parsing/matching and existing tests. Enrichment is available without
an additional feature flag in this implementation. No planned importer feature is advertised.
The guide separates execution creation, cash movements and enrichment, and explains required
versus optional sources, ordering, periods, duplicates, unmatched rows, conflicts, reconstructed
versus reported amounts, and the separate limits of retained-source reprocessing.

Implementation only changes Web guidance, catalogue formats, translations and UI tests.
No parser, accounting, broker P&L, schema or import API changes. Uses the existing accordion.
Shown before selecting a file, from the SBI connection flow and for the canonical SBI account
selected on the generic import page. Independent and repeated uploads remain available.

## Broker export evidence and remaining verification

Read these live official public pages on 2026-10-19; also opened the My資産 guide and its domestic
stock detail page in the Codex built-in browser:

- [SBI product-specific realized-P&L guide](https://search.sbisec.co.jp/v2/popwin/guide/tool/assets/03_profits_loss/profits_loss.html)
  shows 実現損益詳細, 約定日/受渡日, period/account filters and 照会.
- [SBI domestic stock detail and CSV guide](https://search.sbisec.co.jp/v2/popwin/guide/tool/assets/03_profits_loss/profits_loss_domestic.html)
  shows 国内株式（現物） / 国内株式（信用）, CSVダウンロード, prior-day execution availability,
  a 5,000-detail-row download limit, and provisional values before settlement.
- Existing #201 paths remain `口座管理 → 取引履歴 → 約定履歴` and `入出金 → 入出金明細`.

**Pending:** the enabled browser has no authenticated SBI session. These new labels and export
steps are checked against current public SBI UI documentation and supported code, but were
not independently exercised inside a logged-in SBI account. Keep the PR in draft until the
issue's explicit authenticated/live-account UI verification is satisfied. No private exports,
account information or user screenshots were accessed or published.

## Local validation

- Lingui compiled all five catalogues. New prose is translated in English, Simplified Chinese,
  Traditional Chinese (Hong Kong), Japanese and Korean; broker labels remain Japanese.
  Filled the existing Korean/Hong Kong SBI sidebar/summary translation gaps as well.
- `vp check`: passed, zero errors; 350 warnings in the existing repository checks.
- `tsc -b --noEmit`: passed.
- `vp build`: passed.
- Focused Web tests: **32/32** across the guide, ImportView, market locale, import history and
  reprocessing tests. Cover keyboard expansion/collapse, all five translations, account
  switching, upload preservation, default collapsed re-entry, cash preview and existing flows.
- `go test ./internal/importer ./internal/api`: passed both packages, including existing SBI,
  non-SBI, cash/enrichment and retained-source regressions.
- `git diff --check`: passed.

## Real-API Web acceptance

Codex built-in browser against a new SQLite database and this working tree's API on port 8092,
Web on port 5192; synthetic account and CSVs only. Confirmed the active API in Settings and
successfully logged in before evaluating behavior.

- Guide visible before file selection; collapsed initially; Enter expands, Space collapses,
  accessible expanded state reads back correctly.
- Generic import page: switching to non-SBI hides the guide, switching back restores it
  collapsed; no guide-induced preview or upload write.
- Broker connection entry (`/import?broker=sbi`) renders the guide and existing export sidebar.
- Execution CSV preview, return/cancel and re-entry: no executions written on cancellation;
  fresh upload starts empty with the guide collapsed. Re-upload and confirm inserts 2 fills.
- Separately upload and confirm cash CSV: 1 cash movement, no additional executions.
- Separately upload realized-P&L CSV: preview reports 1 enrichable close, apply reports 1
  enriched close; repeat preview reports 1 already enriched row. Return without applying.
- Durable synthetic state read-back: 2 executions, 1 cash transaction, 1 trade, 1 enriched close.
- Desktop rendered guide inspected; 390 × 844 viewport inspected collapsed and expanded in
  Chinese and English, plus Japanese expanded. Guide panel scroll width equals client width
  (311px), without horizontal overflow. Viewport override reset afterward.
- All five language versions opened and the expanded guide read back in the real browser.
- Local screenshot evidence: `/private/tmp/352-desktop.jpg`, `352-narrow-collapsed.jpg`,
  `352-narrow-expanded.jpg`, `352-narrow-en.jpg`. Not uploaded; all visible account data is synthetic.

No iOS/Android validation; outside this Web-first issue. No merge, release, deployment or
production-data changes performed. Remote CI is separate from the local checks above.
