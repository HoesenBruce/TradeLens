import type { Trade, TradeDetail } from "./api/types";
import { gradeFromInt, intFromGrade, type TradeGrade } from "./tradeGrades";

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

export class AmbiguousReviewNotesError extends Error {}

// Keep source offsets: parsing/rebuilding other sections loses freeform plan text.
function reviewSections(raw: string) {
  if (/\r(?!\n)/.test(raw)) return undefined;
  const sections: { start: number; end: number; newline: boolean }[] = [];
  let offset = 0;
  let fence: { marker: string; length: number } | undefined;
  for (const line of raw.match(/[^\n]*\n|[^\n]+$/g) ?? []) {
    const text = line.replace(/\r?\n$/, "");
    const delimiter = /^ {0,3}(`{3,}|~{3,})(.*)$/.exec(text);
    if (fence) {
      if (
        delimiter &&
        delimiter[1][0] === fence.marker &&
        delimiter[1].length >= fence.length &&
        /^[ \t]*$/.test(delimiter[2])
      )
        fence = undefined;
    } else if (delimiter) {
      // Backticks in a backtick info string invalidate the opener. Never use
      // an invalid fence to hide HTML or apparent headings from these checks.
      if (delimiter[1][0] === "`" && delimiter[2].includes("`")) return undefined;
      fence = { marker: delimiter[1][0], length: delimiter[1].length };
    } else {
      // Only flat ATX sections and closed fences have supported boundaries.
      // Do not infer HTML, Setext, container, indented-code or reference contexts.
      if (
        text.includes("<") ||
        /^ {0,3}(?:=+|-+)[ \t]*$/.test(text) ||
        /^ {0,3}(?:>|[-+*](?:[ \t]|$)|\d{1,9}[.)](?:[ \t]|$)|\[.*\]:)/.test(text) ||
        (/^(?: {4}| {0,3}\t)/.test(text) && /[^ \t]/.test(text))
      )
        return undefined;
      const heading = /^ {0,3}(#{1,2})(?:[ \t]+(.*)|$)/.exec(text);
      const target =
        heading?.[1] === "##" &&
        (heading[2] ?? "")
          .replace(/[ \t]+#+[ \t]*$/, "")
          .trim()
          .toLowerCase() === "review notes";
      if (!heading && /^##Review notes[ \t]*$/i.test(text)) return undefined;
      if (!heading) {
        offset += line.length;
        continue;
      }
      const previous = sections.at(-1);
      if (previous?.end === raw.length) previous.end = offset;
      if (target) {
        sections.push({
          start: offset + line.length,
          end: raw.length,
          newline: line.endsWith("\n"),
        });
      }
    }
    offset += line.length;
  }
  return fence || sections.length > 1 ? undefined : sections;
}

export function reviewNotesValue(raw: string) {
  const section = reviewSections(raw)?.[0];
  return section ? raw.slice(section.start, section.end).trim() : "";
}

export function replaceReviewNotes(raw: string, value: string) {
  const sections = reviewSections(raw);
  if (!sections) throw new AmbiguousReviewNotesError();
  const eol = raw.match(/\r\n|\n/)?.[0] ?? "\n";
  const section = sections[0];
  if (!section) {
    if (!value) return raw;
    const separator = !raw || raw.endsWith(eol + eol) ? "" : raw.endsWith(eol) ? eol : eol + eol;
    return raw + separator + "## Review notes" + eol + value;
  }
  const body = raw.slice(section.start, section.end);
  const content = body.trim();
  const start = section.start + (content ? body.indexOf(content) : 0);
  const end = start + content.length;
  const before = section.newline ? "" : eol;
  const after = section.end < raw.length && end === section.end ? eol : "";
  return raw.slice(0, start) + before + value + after + raw.slice(end);
}

export interface ReviewDraft {
  grade: TradeGrade | "";
  notes: string;
  mistakeIds: string[];
}

export function reviewDraft(trade: TradeDetail): ReviewDraft {
  return {
    grade: gradeFromInt(trade.trade_quality),
    notes: reviewNotesValue(trade.notes),
    mistakeIds: trade.tags.filter((tag) => tag.kind === "mistake").map((tag) => tag.id),
  };
}

export function reviewChanges(initial: ReviewDraft, draft: ReviewDraft) {
  const changes: { trade_quality?: number; reviewNotes?: string; mistakeIds?: string[] } = {};
  if (initial.grade !== draft.grade) changes.trade_quality = intFromGrade(draft.grade) ?? 0;
  if (initial.notes !== draft.notes) changes.reviewNotes = draft.notes;
  if (
    initial.mistakeIds.length !== draft.mistakeIds.length ||
    initial.mistakeIds.some((id) => !draft.mistakeIds.includes(id))
  )
    changes.mistakeIds = draft.mistakeIds;
  return changes;
}

export function reviewPatch(trade: TradeDetail, changes: ReturnType<typeof reviewChanges>) {
  const patch: { trade_quality?: number; notes?: string; tag_ids?: string[] } = {};
  if (changes.trade_quality !== undefined) patch.trade_quality = changes.trade_quality;
  if (changes.reviewNotes !== undefined)
    patch.notes = replaceReviewNotes(trade.notes, changes.reviewNotes);
  if (changes.mistakeIds !== undefined) {
    patch.tag_ids = [
      ...new Set([
        ...trade.tags.filter((tag) => tag.kind !== "mistake").map((tag) => tag.id),
        ...changes.mistakeIds,
      ]),
    ];
  }
  return patch;
}

export function backlogScope(user: string, accounts?: string[]) {
  return `tm_review_backlog:${user}:${accounts?.length ? [...accounts].sort().join(",") : "all"}`;
}

export const reviewCutoffKey = (account: string) => `reviewBacklogCutoff:${account}`;
