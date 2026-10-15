import { describe, expect, it } from "vite-plus/test";
import type { TradeDetail } from "./api/types";
import {
  AmbiguousReviewNotesError,
  replaceReviewNotes,
  reviewChanges,
  reviewDraft,
  reviewPatch,
} from "./reviewInbox";

const htmlDocuments = [
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
  ["literal less-than", "Risk < budget\n\n## Review notes\nold"],
] as const;

describe("HTML journal boundaries fail closed", () => {
  for (const eol of ["\n", "\r\n"]) {
    for (const [name, document] of htmlDocuments) {
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
