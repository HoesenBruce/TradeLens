import { describe, expect, it } from "vite-plus/test";
import type { Trade, TradeDetail } from "./api/types";
import { backlogScope, isReviewed, reviewPatch, reviewQueue } from "./reviewInbox";
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
    const patch = reviewPatch(saved, 5, "new", ["new"]);
    expect(Object.keys(patch).sort()).toEqual(["notes", "tag_ids", "trade_quality"]);
    expect(parseJournalNotes(patch.notes)).toEqual({ ...journal, reviewNotes: "new" });
    expect(patch.tag_ids).toEqual(["tag", "emotion", "new"]);
    expect(saved).toEqual(before);
    expect(
      parseJournalNotes(reviewPatch({ ...saved, notes: "plain legacy" }, 3, "review", []).notes)
        .legacy,
    ).toBe("plain legacy");
    expect(reviewPatch(saved, 3, "old", []).notes).toBe(saved.notes);
  });
  it("scopes dismissal by user and exact account set", () => {
    expect(backlogScope("u", ["b", "a"])).toBe(backlogScope("u", ["a", "b"]));
    expect(backlogScope("u", ["a"])).not.toBe(backlogScope("v", ["a"]));
    expect(backlogScope("u", ["a"])).not.toBe(backlogScope("u"));
  });
});
