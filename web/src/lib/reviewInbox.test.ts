import { describe, expect, it } from "vite-plus/test";
import type { Trade, TradeDetail } from "./api/types";
import {
  AmbiguousReviewNotesError,
  backlogScope,
  isReviewed,
  replaceReviewNotes,
  reviewChanges,
  reviewDraft,
  reviewNotesValue,
  reviewPatch,
  reviewQueue,
} from "./reviewInbox";
import { buildStructuredJournalNotes, parseJournalNotes } from "./journalNotes";
const now = Date.parse("2026-10-10T12:00:00Z");
const trade = (id: string, days: number, quality?: number): Trade =>
  ({
    id,
    closed_at: new Date(now - days * 86400000).toISOString(),
    trade_quality: quality,
  }) as Trade;
describe("review inbox", () => {
  it("uses grades alone, exact 14-day boundary, excludes open and future trades", () => {
    const trades = [
      trade("recent", 1),
      trade("boundary", 14),
      trade("old", 15),
      trade("graded", 1, 3),
      trade("future", -1),
      { ...trade("open", 1), closed_at: null },
    ];
    expect(reviewQueue(trades, 14, false, now).map((t) => t.id)).toEqual(["boundary", "recent"]);
    expect(reviewQueue(trades, 14, true, now).map((t) => t.id)).toEqual(["old"]);
    for (const trade_quality of [undefined, null, 0, 6])
      expect(isReviewed({ trade_quality })).toBe(false);
  });
  it("patches only review-owned fields, preserving planned direction, legacy, setups and non-mistake tags", () => {
    const journal = {
      plannedDirection: "short" as const,
      session: "Asia",
      entryReason: "entry",
      exitReason: "exit",
      reviewNotes: "old",
      legacy: "## Custom\nkeep",
    };
    const saved = {
      direction: "long",
      notes: buildStructuredJournalNotes(journal),
      tags: [
        { id: "tag", kind: "tag" },
        { id: "old", kind: "mistake" },
        { id: "emotion", kind: "emotion" },
      ],
      setup_ids: ["a", "b"],
      target_price: 200,
      initial_risk: 100,
      stop_price: 90,
      confidence: 4,
    } as TradeDetail;
    const before = structuredClone(saved);
    const patch = reviewPatch(saved, { trade_quality: 5, reviewNotes: "new", mistakeIds: ["new"] });
    expect(Object.keys(patch).sort()).toEqual(["notes", "tag_ids", "trade_quality"]);
    expect(parseJournalNotes(patch.notes!)).toEqual({ ...journal, reviewNotes: "new" });
    expect(patch.tag_ids).toEqual(["tag", "emotion", "new"]);
    expect(saved).toEqual(before);
    expect(
      parseJournalNotes(
        reviewPatch(
          { ...saved, notes: "plain legacy" },
          { trade_quality: 3, reviewNotes: "review" },
        ).notes!,
      ).legacy,
    ).toBe("plain legacy");
    expect(reviewPatch(saved, { trade_quality: 3 })).toEqual({ trade_quality: 3 });
  });
  it("scopes dismissal by user and exact account set", () => {
    expect(backlogScope("u", ["b", "a"])).toBe(backlogScope("u", ["a", "b"]));
    expect(backlogScope("u", ["a"])).not.toBe(backlogScope("v", ["a"]));
    expect(backlogScope("u", ["a"])).not.toBe(backlogScope("u"));
  });
});

describe("lossless review notes", () => {
  it("recognizes indented headings and preserves neighboring sections", () => {
    const prefix = "legacy\n  ## Review notes ###  \n";
    const suffix = "\n\n   ## Unknown\nuntouched\n";
    expect(replaceReviewNotes(prefix + "old" + suffix, "new")).toBe(prefix + "new" + suffix);
    expect(() => replaceReviewNotes("```md\n## Review notes\nexample", "new")).toThrow(
      AmbiguousReviewNotesError,
    );
  });
  for (const eol of ["\n", "\r\n"]) {
    for (const plan of ["short", "SHORT", "short\nReason: hedge existing exposure"]) {
      it(`preserves raw content: ${JSON.stringify({ eol, plan })}`, () => {
        const prefix =
          "  legacy  \n\n## Planned direction\n" +
          plan +
          "\n\n## Unknown\n keep  \n\n## Review notes  \n\n  ";
        const suffix = "  \n\n \n## Exit reason\n  untouched \n\n";
        const raw = (prefix + "old" + suffix).replaceAll("\n", eol);
        expect(replaceReviewNotes(raw, "new")).toBe(
          (prefix + "new" + suffix).replaceAll("\n", eol),
        );
      });
    }
  }
  it("appends without changing existing source bytes", () => {
    for (const raw of ["", "freeform  ", "## Unknown\nkeep\n", "legacy\r\n\r\n"]) {
      const saved = replaceReviewNotes(raw, "new");
      expect(saved.startsWith(raw)).toBe(true);
      expect(reviewNotesValue(saved)).toBe("new");
      expect(replaceReviewNotes(raw, "")).toBe(raw);
    }
  });
  it("handles empty/EOF sections, higher headings, nested content and clearing", () => {
    expect(replaceReviewNotes("## Review notes", "new")).toBe("## Review notes\nnew");
    expect(replaceReviewNotes("## Review notes\n## Next\nkeep", "new")).toBe(
      "## Review notes\nnew\n## Next\nkeep",
    );
    expect(replaceReviewNotes("## Review notes\nold\n\n# Next\nkeep", "")).toBe(
      "## Review notes\n\n\n# Next\nkeep",
    );
    expect(replaceReviewNotes("## Review notes\nold\n### Nested\ncontent\n\n## Next", "new")).toBe(
      "## Review notes\nnew\n\n## Next",
    );
  });
  it("ignores headings inside fenced examples", () => {
    for (const fence of ["```", "~~~~"]) {
      const prefix = `${fence}md\n## Review notes\nexample\n${fence}\n\n`;
      expect(replaceReviewNotes(prefix, "new")).toBe(prefix + "## Review notes\nnew");
      expect(replaceReviewNotes(prefix + "## Review notes\nold", "new")).toBe(
        prefix + "## Review notes\nnew",
      );
      expect(replaceReviewNotes("## Review notes\nold\n" + prefix + "## Other\nkeep", "new")).toBe(
        "## Review notes\nnew\n\n## Other\nkeep",
      );
    }
  });
  it("rejects repeated headings while allowing grade-only saves", () => {
    const notes = "## Review notes\nfirst\n\n## REVIEW NOTES\nsecond";
    expect(() => replaceReviewNotes(notes, "new")).toThrow(AmbiguousReviewNotesError);
    expect(reviewPatch({ notes } as TradeDetail, { trade_quality: 5 })).toEqual({
      trade_quality: 5,
    });
  });
});

describe("field-level review changes", () => {
  const initial = {
    notes: "## Planned direction\nSHORT",
    trade_quality: null,
    tags: [],
  } as unknown as TradeDetail;
  const baseline = reviewDraft(initial);
  const latest = {
    ...initial,
    notes: initial.notes + "\n\n## Review notes\nnew remote note",
    trade_quality: 3,
    tags: [
      { id: "remote-mistake", kind: "mistake" },
      { id: "remote-custom", kind: "custom" },
    ],
  } as TradeDetail;
  it("omits untouched notes and tags on grade-only saves", () => {
    expect(reviewPatch(latest, reviewChanges(baseline, { ...baseline, grade: "A+" }))).toEqual({
      trade_quality: 5,
    });
  });
  it("omits a remote grade and tags when only notes change", () => {
    expect(reviewPatch(latest, reviewChanges(baseline, { ...baseline, notes: "local" }))).toEqual({
      notes: initial.notes + "\n\n## Review notes\nlocal",
    });
  });
  it("merges dirty mistakes with latest unrelated tags", () => {
    expect(
      reviewPatch(latest, reviewChanges(baseline, { ...baseline, mistakeIds: ["local-mistake"] })),
    ).toEqual({ tag_ids: ["remote-custom", "local-mistake"] });
  });
  it("keeps reverted values and reordered sets untouched", () => {
    expect(reviewChanges(baseline, { ...baseline })).toEqual({});
    expect(
      reviewChanges(
        { ...baseline, mistakeIds: ["a", "b"] },
        { ...baseline, mistakeIds: ["b", "a"] },
      ),
    ).toEqual({});
  });
});
