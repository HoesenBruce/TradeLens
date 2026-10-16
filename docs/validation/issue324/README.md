# #324 acceptance

Base: `7aad3b43`. Foundation only; business rollout follows separately in #325.

- Component/catalog unit tests: passed (2 tests); every definition field, five locales,
  interpolation, unsupported-locale fallback, Enter, Escape, close and focus restoration.
  Focused holding/quantity/i18n regressions: 5 files / 12 tests passed.
- Playwright: 2 passed, 1440px and 390px. Five locales each, localized close control,
  Enter/Space, touch tap, Escape, outside click, close, reopening and horizontal bounds.
- Built-in browser: desktop and 390px visual inspection and keyboard/click interactions.
- Static check: 0 errors (347 existing warnings). TypeScript and production build passed.
- Disposable API `/healthz`: working, v0.4.0. The isolated component fixture does not
  request account data; no cached finance values are used as acceptance evidence.
- Full local unit run was interrupted after unrelated NewTradeDrawer timeout / stalled
  progress; it is not claimed as a pass. NewTradeDrawer rerun alone: 21 passed.
  PR CI must pass before merge.
- Read-through review: no financial computation, server, stored-data or account access
  changes. Optional breakdown slot/document link only; no Level 2/3 feature implementation.

Screenshots: [desktop](desktop-en.png), [Chinese narrow](narrow-zh-CN.png).
Mobile apps not validated; outside current fork scope. No release or deployment.
