# Issue #200 — login logo size

The existing logo artwork is unchanged. AuthShell now displays the desktop
brand icon at 48px (previously 30px) and the centered narrow-layout icon at
72px (previously 36px), using the existing 256px asset. Wordmark, tagline,
spacing, form structure, and other AppLogo consumers are unchanged.

## Validation

- `pnpm check`: passed, 0 errors; 340 existing warnings.
- `pnpm test src/components/AuthShell.test.tsx`: passed (1 test).
- Codex built-in browser against a throwaway SQLite API on port 8099 and
  working-tree Web server on port 5190, with CORS enabled for that origin.
- Desktop 1280px, narrow 390px, and 320px: inspected rendered layout;
  logo sharp and aligned, no clipping/overlap or horizontal overflow.
  Desktop form remains in its original column; narrow form retains its
  hierarchy with the enlarged centered header.
- English → Japanese → English: labels and selected language reflected the
  current choice; branding remained unchanged.
- Invalid credentials displayed the API error; valid test credentials opened
  Home. Signing out and reloading returned to the enlarged login header.
- This presentation-only change adds no cancel/reset or toggle states.
- CI skipped at user request. Mobile app not validated; outside fork scope.

Screenshots: [desktop](login-desktop.png), [390px](login-390.png).
