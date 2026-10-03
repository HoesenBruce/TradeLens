# Versioned session evidence

CSV snapshots from `exchange-calendars==4.13.2`, XTKS (JP) / XNYS (US),
2020-01-01 through 2030-12-31. Generate with `api/scripts/export_calendars.py`.
The Go runtime uses these local snapshots and has no Python/calendar-network dependency.
Outside the snapshot's bounds validation reports `calendar_unavailable`.
Update snapshots and CalendarVersion together; historical evidence keeps its version.
Future exceptional closures require a reviewed snapshot update.

Source: https://github.com/gerrymanoim/exchange_calendars (Apache-2.0).
Spot checks: JP 2026-09-21 through 23 holidays, post-2024-11-05 15:30 close;
US DST and 2026-11-27 13:00 early close. Official exchange references:
https://www.jpx.co.jp/english/corporate/about-jpx/calendar/
https://www.jpx.co.jp/english/equities/trading/domestic/01.html
https://www.nyse.com/trade/hours-calendars
These are exchange calendar rules, not assertions that every stock traded.

## Output rights and provenance

The Apache-2.0 designation above describes the generator library, not a license
assigned to `JP.csv` or `US.csv`. The exporter writes only calculated session
dates and opening/closing timestamps; no library source or documentation is
vended in those CSVs. Retain this generation/version provenance when updating
the snapshots. See [the third-party review](../../../../THIRD_PARTY_NOTICES.md#generated-exchange-calendar-output).
