# PR #292 independent-review remediation

Starting Head: `7312425c3773ae5de07db3a712c1e7566c3fe2a8`.
Base/main: `b63aead3074e08435979a93c6aa84acbe021d189`, including #310.

The Inbox edits only the located Review-notes body in the latest saved journal.
All source slices outside that body, including blank lines, spaces, CRLF,
unknown sections and the complete raw Planned-direction section, are retained.
Missing sections are appended without rewriting existing bytes. Markdown fenced
examples do not become section boundaries. Multiple Review-notes headings or
an unclosed code fence reject a notes edit before PATCH, retain the draft, and
show a localized explanation. Grade-only saves can still preserve such documents.
No accounting direction or plannedDirection parser semantics are changed.

Each editor keeps its opening baseline separately from its draft. Grade, notes
and the set of mistake IDs are compared independently; reverting a field makes
it untouched again. Save refreshes the server trade and sends only changed
fields. A changed mistake selection uses the latest non-mistake tags. A changed
note uses the latest raw journal. Cancel adopts the current persisted snapshot;
failed saves retain the original baseline and edited draft, so retry refreshes
again without making untouched fields dirty.

## Concurrent-edit limits

This is field-level stale-editor protection, not strict conflict safety. There
is no revision/ETag conditional update or atomic compare-and-swap. Another writer
can commit between the refresh GET and PATCH; changes to the same review field
remain last-writer-wins. Notes and tag memberships are full replacement fields,
so an intervening write to either can be overwritten by an edit to that field.
The inherited API journal handler also merges and upserts without a transactional
version check; truly simultaneous PATCH requests can race even for different
journal fields. These residual races are not claimed fixed by this change.

## Regression evidence

- `web/src/lib/reviewInbox.test.ts`: raw uppercase/lowercase/explained plans,
  exact whitespace and LF/CRLF preservation, unknown sections, absent/empty/EOF
  sections, nested/higher/indented headings, fences, repeated/ambiguous sections,
  grade-only payloads, notes-only/mistakes-only payloads and reverted fields.
- `web/e2e/review-remediation.spec.ts`: browser editor A plus a separately logged-in
  HTTP editor B. Six raw-plan/newline combinations and absent-section insertion
  are checked byte-for-byte through real API persistence/readback. Independent
  grade-only, notes-only, mistakes-only and failed-save/retry cases verify exact
  PATCH keys and current server values. Repeated headings issue no PATCH and
  retain the full stored document and local draft; cancel then grade-only saves.
- Existing Review Inbox, SBI closing-first/direction/reverse-import, #310
  quantity rendering and the unchanged 35-case production security matrix
  remain required acceptance gates. Security assertions, deadlines, zero retries
  and Worker repeatEach=3 are unchanged.

Final commit identity, executed commands, complete results, CI provenance and
second acceptance review are recorded in the PR Conversation evidence comment.
No historical SBI regroup, migration, release or deployment is part of this work.
PR remains Draft. Mobile and optional PostgreSQL integration are outside the
configured local verification environment and must not be inferred as covered.
