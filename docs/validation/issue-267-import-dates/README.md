# Issue 267 Web acceptance

2026-10-09; Codex built-in browser, isolated working-tree API :8093 and Web
:5178, disposable SQLite database. All CSV contents and the QA account are synthetic.
The Web server was explicitly pointed at the live QA API before acceptance.

- Ambiguous 05/01 and 06/01 dates initially block confirmation and show both readings.
- DD/MM refresh shows January 5/6; switching to MM/DD blocks confirmation until
  refresh and shows May 1/June 1 instead. Reopening the control preserves its selection.
- Tokyo timezone refresh shifts those timestamps to 00:30/06:59 UTC.
- Returning without committing leaves zero batches and executions. Re-entry resets
  abandoned choices to UTC and unresolved date order.
- DD/MM + UTC confirmation writes two executions. API read-back exactly matches
  preview: 2026-01-05T09:30:00Z and 2026-01-06T15:59:00Z.
- Identical reimport writes zero executions and skips two duplicates.
- SBI YYYY/MM/DD preview shows no slash-date prompt and labels its synthetic
  2026-09-01T00:00:00Z timestamp as date-only.

Screenshots show the reviewed DD/MM preview, initial commit, duplicate reimport,
and SBI preview. Mobile was not validated; outside current fork scope.
