# Issue 201 validation

- Local Web production build and `pnpm check` passed (0 errors; 340 existing warnings).
- Broker/i18n regression: 13 tests passed. Go importer tests passed.
- Codex built-in browser against disposable SQLite API (`SBI QA`): English, Japanese, and Simplified Chinese preserve both official Japanese menu paths and CSV labels; surrounding instructions translate.
- Desktop and 390px rendered layouts inspected, including the guide below the upload area.
- Execution-history CSV alone reaches SBI auto-mapped preview. Back returns to an empty upload surface with the guide intact.
- Cash-history CSV alone reaches ledger preview and commits 4 transactions; the fixture's intentionally invalid fifth date is reported as a row error. Import another returns to the upload surface.
- No simultaneous-upload requirement, parser, accounting, or P&L changes.
- CI skipped by request. Mobile not validated; outside current fork scope.
