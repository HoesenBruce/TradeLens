# Rolling startup-date showcase — 2026-10-08

This follow-up to #243 / PR #264 replaces fixed Render sample dates with the
30 days preceding each service start, based on the startup date in Asia/Tokyo.
It does not regenerate data on every click or warm request. A visitor waking a
sleeping instance receives a freshly seeded window after readiness succeeds.

One shared timeline maps the six ordered showcase events onto actual Japanese
sessions from the existing JP calendar snapshot. Transactions, deposits and
withdrawals, news publication dates and fictional daily bars use that timeline.
Intervening sessions use the previous fictional event price. No live prices or
private data are involved. The runtime freezes the date before launching any
child, avoiding a seed/provider disagreement around midnight.

The seven trade groups and monetary results remain unchanged: JPY 25,000
realized P&L, JPY 950,000 contributed and JPY 975,000 closing value. Holding
durations, weekday breakdowns and calendar dates naturally change with the new
distribution. Prediction creation remains at startup; expected coverage stays
four pending / two unavailable, with no claim of validated predictions.

Standalone showcase commands remain fixed/reproducible by default. The new
timeline is selected only by the demo runtime's shared startup-date environment.
Normal self-hosted Docker/Compose, authentication and demo read-only policy are
unchanged. The JP calendar must be extended before its existing coverage ends;
unsupported startup dates fail rather than guessing holiday sessions.

## Validation

- `python3 scripts/test_showcase.py`: PASS. Fixed legacy/showcase output,
  rolling ranges, year/month boundaries, leap-year input, Japanese holidays,
  out-of-calendar rejection, next-day date movement, matching CSV/bar dates,
  no bars outside generated coverage, and the existing Web E2E importlib caller.
- Demo Docker build: PASS, local Linux ARM64.
- `python3 scripts/test_demo_render.py`: PASS; [output](rolling-container-test.txt).
  Two boots under 512 MiB / 0.5 CPU verify rolling transaction/news dates,
  entry range replacement, complete daily valuations, duplicate-free reset,
  stale authentication rejection, permissions, SBI and news outcomes.
- Codex built-in browser against the final local container/API: PASS. Entry
  displayed **2026-09-08 – 2026-10-07** for an October 8 Tokyo startup. Login
  opened Home with **both default 30-day charts populated**, without selecting
  All. Both chart ranges changed 30→90→30; refresh returned populated default
  charts, and reopening `/demo` retained the same running-instance range.
  Screenshots: [entry](rolling-entry.jpg), [default Home charts](rolling-home.jpg).

## Remaining public acceptance

This new runtime has **not yet been redeployed or tested on Render**. The earlier
[public acceptance](public-render-acceptance.md) applies to the previous fixed-date
implementation. Owner must deploy the new PR head, then check the displayed
startup-relative window, default Home charts, SBI/news results and cold-start
recovery again. Dashboard Free/no-paid-resource/deployed-SHA evidence remains
outstanding. Native mobile was not tested; outside the current fork scope.
