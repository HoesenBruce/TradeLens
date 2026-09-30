# Public financial fixture provenance

Issue #210 replaces unproved static samples. No original export is an input.
Report headers and parser edge conditions are format specifications, not
preserved account activity.

Run `python3 scripts/generate-financial-fixtures.py` to write eleven files;
run with `--check` to verify byte-for-byte reproducibility. Python's standard
library suffices, including for the deterministic XLSX archive.

| Outputs | Independent source / coverage |
| --- | --- |
| `api/internal/importer/testdata/sbi-*.csv` | Invented 2031 dates, 910x/73xA symbols and quantities/prices/settlements. Cash/margin long/short, date ordering, duplicate fills, invalid quantity/date, 現引, combined name/code, signed settlement and invalid margin rows. |
| `api/internal/importer/testdata/mt*-statement.*` | Newly constructed HTML/OOXML grids with invented ticket IDs, AUDUSD/EURJPY/gold fills and costs. Balance/order skipping, malformed volume/price, NBSP price, EET/UTC and numeric Excel timestamp paths. |
| `api/testdata/generic_sample.csv`, `web/public/sample-*` | Invented SYNTH 19-share round trip at 73/79, malformed generic quantity. JSON P&L is calculated from fills. |
| `docs/demo/tradermemos-demo-trades.json` | Procedural `seed-demo.py` algorithm, RNG seed 210, fixed 2031-02-28 UTC, plus invented partial-close. Stock/option ×100, long/short and setups. |

`seed-demo.py` uses hardcoded illustration targets/reference prices and a local
seeded RNG, with no broker/file/market-data input. `--end-date` fixes its date
window for screenshots. API calls seed data; they are not a source of trading
records for the generator.

Current-tree inventory: tracked financial CSV/JSON/HTML/XLSX datasets are these
eleven outputs. JP/US calendar CSVs are exchange calendars, not account records.
Inline Go/TS tests and E2E seed helpers construct arithmetic cases in source;
existing split, 現渡, partial-close and non-SBI tests remain. Removed journal
exports remain removed. Planning documents contain illustrative code, not
runtime fixtures. Screenshots are #211; historical blobs and the final
publication gate remain #207 follow-ups.
