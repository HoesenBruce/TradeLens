import type { Trade, TradeDetail } from "./api/types";
import { buildStructuredJournalNotes, parseJournalNotes } from "./journalNotes";

export function isReviewed(trade: Pick<Trade, "trade_quality">): boolean {
  return trade.trade_quality != null && trade.trade_quality >= 1 && trade.trade_quality <= 5;
}

export function reviewQueue(trades: Trade[], days: number, backlog: boolean, now = Date.now()) {
  const cutoff = now - days * 86400000;
  return trades
    .filter((t) => {
      if (!t.closed_at || isReviewed(t)) return false;
      const closed = Date.parse(t.closed_at);
      return closed <= now && (backlog ? closed < cutoff : closed >= cutoff);
    })
    .sort(
      (a, b) => Date.parse(a.closed_at!) - Date.parse(b.closed_at!) || a.id.localeCompare(b.id),
    );
}

export function reviewPatch(
  trade: TradeDetail,
  grade: number,
  reviewNotes: string,
  mistakeIds: string[],
) {
  const journal = parseJournalNotes(trade.notes);
  return {
    trade_quality: grade,
    notes:
      reviewNotes === journal.reviewNotes
        ? trade.notes
        : [journal.legacy, buildStructuredJournalNotes({ ...journal, legacy: "", reviewNotes })]
            .filter(Boolean)
            .join("\n\n"),
    tag_ids: [
      ...new Set([
        ...trade.tags.filter((t) => t.kind !== "mistake").map((t) => t.id),
        ...mistakeIds,
      ]),
    ],
  };
}

export function backlogScope(user: string, accounts?: string[]) {
  return `tm_review_backlog:${user}:${accounts?.length ? [...accounts].sort().join(",") : "all"}`;
}

export const reviewCutoffKey = (account: string) => `reviewBacklogCutoff:${account}`;
