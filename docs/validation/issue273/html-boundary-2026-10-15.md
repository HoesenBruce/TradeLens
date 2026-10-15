# PR #292 HTML Journal boundary remediation

Starting Head: `5a841f4242e80443e752de43d20b0977820bb18e`.
Base/main: `b63aead3074e08435979a93c6aa84acbe021d189` (#310 retained).
Independent blocker: PR #292 comment `6185280151`.

## Conservative policy

The Inbox does not parse HTML. If any line outside a Markdown code fence contains
`<`, the whole Journal is treated as ambiguous for notes editing. This includes
well-formed, closed, uppercase, mixed-case, attributed, multiline, nested,
malformed and unclosed HTML; it also deliberately includes literal comparisons,
escaped less-than characters and other non-HTML uses of `<` outside a fence.
There is no tag allowlist, nesting heuristic, or inferred safe HTML end boundary.
Even a real Review-notes heading after a closed HTML block remains protected by
this policy. Notes editing fails before PATCH, displays the existing localized
ambiguity message and retains the draft. Cancel remains available.

No apparent Review-notes body from such a document is loaded into the editor.
Grade-only changes still send only `trade_quality`; the stored Journal, including
all HTML, closing tags, whitespace, LF/CRLF and other sections, remains untouched.
Mistake-only changes likewise do not rewrite notes. An untouched HTML Journal
that another editor added after opening is also preserved by a grade-only save;
a dirty notes edit against it is rejected when the latest document is fetched.

Only flat ATX sections, plain text and known closed Markdown fences have
supported boundaries. Backtick fence info strings containing backticks are
invalid and reject the whole notes edit. Outside fences, possible Setext
underlines (`=`/`-` lines), list/blockquote markers, indented code and link-reference
definitions also reject editing rather than attempting another partial parser.
Fence closing tails accept ASCII space/tab only, so NBSP, vertical tab and form
feed cannot turn an unclosed example into editable sections. Whitespace-only
indented blank lines remain verbatim. Bare CR line endings reject editing; LF and CRLF remain supported. These rules
are deliberately conservative, including horizontal rules and incidental matching
text. Repeated Review headings, unclosed fences and the nonstandard legacy
`##Review notes` candidate (missing ATX whitespace) likewise reject. Grade-only
PATCH remains available in every rejected case.

HTML inside known closed, valid Markdown fences stays literal code. Lossless
slice replacement/append remains in use for supported,
unambiguous documents. Unknown sections, complete Planned-direction contents,
legacy text and raw whitespace outside the edited body remain source slices.
No common Journal serializer or accounting/plannedDirection semantics change.

This is intentionally restrictive rather than a general Markdown/HTML parser.
No new API endpoint, migration, dependency or concurrency mechanism is added.
GET/PATCH windows and simultaneous API read/merge/upsert races remain as described
in `remediation-2026-10-15.md`; this change does not claim strict conflict safety.

## Permanent regression evidence

- `web/src/lib/reviewInbox.html.test.ts`: 74 cases covering the exact independent
  reproduction, PRE case variants, attributes/multiline tags, nested/malformed/
  unclosed blocks, orphan closing tags, repeated apparent headings, real headings
  following HTML, no real target, comments/other HTML and literal `<`; LF/CRLF,
  grade-only payloads, no object mutation, and preserved backtick/tilde fences.
  Includes the independent reviewer's invalid-backtick-info HTML bypass and
  exact closing-tag loss, Setext H1/H2, containers, references and bare CR.
- `web/e2e/review-html-boundary.spec.ts`: independent authenticated clients plus
  the actual production Inbox. For 24 raw documents in LF and CRLF, notes edits
  issue zero PATCHes, preserve the draft and persisted full detail, then Cancel
  plus grade-only PATCH preserves all original bytes and unrelated API fields.
  Four valid empty-ATX-boundary edits preserve the following section and full
  unrelated detail. Eight closed-fence edit/append scenarios retain HTML examples verbatim and
  verify persisted GET from the independent client. A separate stale-editor case
  adds HTML via the other client after opening: dirty notes reject, while a
  subsequent grade-only save preserves that newest document.
- Before the correction, the permanent unit regression had 30 failures / 4
  existing fence controls passing. The permanent production browser regression
  failed on the exact original fixture and reported
  `PERSISTED_ORIGINAL_UNCHANGED=false` after an independent API GET. These expected
  red results are retained rather than counted as successful validation.
- Separate read-only acceptance of the first local correction found invalid
  backtick-info and Setext blockers through independent API persistence tests.
  The extended permanent unit suite then failed 18 new cases (34 old cases
  passed) before the second correction. A further independent adversarial probe
  found non-ASCII fence-tail acceptance; six new failing cases were captured
  before correcting closure validation to ASCII space/tab only. These findings were fixed before any
  remediation push. A subsequent independent review found empty ATX H1/H2
  boundaries were not recognized: eight permanent unit cases failed before
  their correction, and now preserve those sections verbatim. Final independent
  acceptance is required on the new Head.

Final-source validation results, commands, exact commit identity, original
security-matrix provenance, separate independent review and evidence hashes
are recorded in the PR Conversation. No historical SBI regroup or deployment
is part of this remediation. PR remains Draft pending explicit merge permission.
