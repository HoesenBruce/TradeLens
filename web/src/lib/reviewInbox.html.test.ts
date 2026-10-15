import { describe, expect, it } from "vite-plus/test";
import type { TradeDetail } from "./api/types";
import {
  AmbiguousReviewNotesError,
  replaceReviewNotes,
  reviewChanges,
  reviewDraft,
  reviewPatch,
} from "./reviewInbox";

const ambiguousDocuments = [
  ["Unicode line separator heading", "## Review notes\u2028not the target\nkeep original"],
  ["Unicode paragraph separator heading", "## Review notes\u2029not the target\nkeep original"],
  [
    "Unicode line separator fence",
    "```html\u2028more\n## Review notes\ncode sample\n```\nplain keep\n```",
  ],
  [
    "Unicode paragraph separator fence",
    "```html\u2029more\n## Review notes\ncode sample\n```\nplain keep\n```",
  ],
  ["independent reproduction", "<pre>\n## Review notes\nexample text\n</pre>\n\n## Unknown\nkeep"],
  ["uppercase", "<PRE>\n## Review notes\nexample\n</PRE>"],
  ["mixed case", "<PrE>\n## Review notes\nexample\n</pRe>"],
  ["attributes", '<pre class="example" data-value=">">\n## Review notes\n  example  \n</pre>'],
  ["multiline opening tag", '<pre\n class="example"\n>\n## Review notes\nexample\n</pre>'],
  ["nested", "<pre>\n<pre>\n## Review notes\nexample\n</pre>\n</pre>"],
  ["malformed", "<pre class=\n## Review notes\nexample\n</PRE"],
  ["unclosed", "<pre>\n## Review notes\nexample"],
  ["orphan closing tag", "## Review notes\nold\n</pre>\n\n## Unknown\nkeep"],
  [
    "multiple apparent headings",
    "<pre>\n## Review notes\nexample\n## Review notes\nexample two\n</pre>",
  ],
  ["real heading after HTML", "<pre>\n## Review notes\nexample\n</pre>\n\n## Review notes\nreal"],
  ["HTML without apparent heading", "<pre>\nexample only\n</pre>\n\n## Planned direction\nSHORT"],
  ["comment", "<!--\n## Review notes\nexample\n-->\n\n## Unknown\nkeep"],
  ["other HTML", "<textarea>\n## Review notes\nexample\n</textarea>"],
  [
    "nonstandard legacy heading",
    "Legacy paragraph\n##Review notes\nnot an actual Markdown heading\n## Unknown\nkeep",
  ],
  ["literal less-than", "Risk < budget\n\n## Review notes\nold"],
  [
    "invalid backtick info bypass",
    '```bad`\n<PRE class="sample">\n## Review notes\nexample text\n</PRE>\n```\n## Review notes\nunrelated code sample\n## Unknown\nkeep',
  ],
  [
    "invalid backtick info closing-tag loss",
    "```bad`\n<pre>\n```\n## Review notes\nexample text\n```bad`\n</pre>\n```\n## Unknown\nkeep",
  ],
  [
    "Setext H1 after notes",
    "## Review notes\nold\n\nUnknown section\n==============\nkeep exact original",
  ],
  [
    "Setext H2 after notes",
    "## Review notes\nold\n\nUnknown section\n--------------\nkeep exact original",
  ],
  ["list container", "- sample\n  ## Review notes\n  unrelated example"],
  ["blockquote container", "> sample\n## Review notes\nold"],
  ["indented code", "    example\n## Review notes\nold"],
  ["reference definition", '[example]: /url "title\n## Review notes\nexample"'],
  [
    "NBSP fence tail",
    "```\n<pre>\n```\u00a0\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
  ],
  [
    "vertical tab fence tail",
    "```\n<pre>\n```\u000b\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
  ],
  [
    "form feed fence tail",
    "```\n<pre>\n```\u000c\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
  ],
  [
    "tilde NBSP fence tail",
    "~~~\n<pre>\n~~~\u00a0\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
  ],
  ["bare carriage return", "## Review notes\nold\r## Unknown\rkeep"],
] as const;

describe("unsupported journal boundaries fail closed", () => {
  for (const eol of ["\n", "\r\n"]) {
    for (const [name, document] of ambiguousDocuments) {
      it(`${name}, ${JSON.stringify(eol)}`, () => {
        const raw = ("  Legacy  \n\n" + document + "\n\n## Unknown tail\n  keep  \n").replaceAll(
          "\n",
          eol,
        );
        const trade = {
          notes: raw,
          tags: [],
          trade_quality: null,
          direction: "long",
        } as unknown as TradeDetail;
        const before = structuredClone(trade);
        const baseline = reviewDraft(trade);
        expect(baseline.notes).toBe("");
        expect(() => replaceReviewNotes(raw, "new")).toThrow(AmbiguousReviewNotesError);
        expect(() =>
          reviewPatch(trade, reviewChanges(baseline, { ...baseline, notes: "new", grade: "A+" })),
        ).toThrow(AmbiguousReviewNotesError);
        expect(reviewPatch(trade, reviewChanges(baseline, { ...baseline, grade: "A+" }))).toEqual({
          trade_quality: 5,
        });
        expect(trade).toEqual(before);
      });
    }
  }

  for (const eol of ["\n", "\r\n"]) {
    for (const space of ["\u00a0", "\u000b", "\u000c", "\ufeff"]) {
      it(`preserves a non-target Unicode title ${JSON.stringify(space)}, ${JSON.stringify(eol)}`, () => {
        const raw = "## Review notes" + space + eol + "keep original";
        expect(replaceReviewNotes(raw, "new")).toBe(
          raw + eol + eol + "## Review notes" + eol + "new",
        );
      });
    }
    for (const marker of ["#", "##"]) {
      for (const indent of ["", "   "]) {
        it(`preserves empty ${marker} section, ${JSON.stringify(indent)}, ${JSON.stringify(eol)}`, () => {
          const prefix = "## Review notes" + eol;
          const suffix = eol + eol + indent + marker + eol + "keep unrelated  " + eol;
          expect(replaceReviewNotes(prefix + "old" + suffix, "new")).toBe(prefix + "new" + suffix);
        });
      }
    }
    for (const [open, close] of [
      ["```html", "```` \t"],
      ["~~~~info`", "~~~~ \t"],
    ]) {
      it(`accepts valid info/closing fences ${JSON.stringify(open)}, ${JSON.stringify(eol)}`, () => {
        const prefix =
          `${open}\n<pre>\n## Review notes\nexample\n</pre>\n${close}\n\n    \t\n`.replaceAll(
            "\n",
            eol,
          );
        expect(replaceReviewNotes(prefix + "## Review notes" + eol + "old", "new")).toBe(
          prefix + "## Review notes" + eol + "new",
        );
      });
    }
    for (const fence of ["```", "~~~~"]) {
      it(`preserves HTML inside a closed ${fence} fence, ${JSON.stringify(eol)}`, () => {
        const prefix =
          `  Legacy  \n\n${fence}html\n<PrE class="example">\n## Review notes\n  sample  \n</pRe>\n${fence}\n\n## Planned direction\nSHORT\nexplanation\n\n`.replaceAll(
            "\n",
            eol,
          );
        const raw =
          prefix + "## Review notes" + eol + "old" + eol + eol + "## Unknown" + eol + " keep  ";
        const expected = raw.replace(eol + "old" + eol, eol + "new" + eol);
        expect(replaceReviewNotes(raw, "new")).toBe(expected);
        expect(replaceReviewNotes(prefix, "new")).toBe(prefix + "## Review notes" + eol + "new");
      });
    }
  }
});
