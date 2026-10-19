import type { Trade } from "./api/types";
import { isLongDirection } from "./tradeDirection";

export type TradeStatusFilter = "win" | "loss" | "open" | "wash";

export function tradeOutcome(t: Trade): TradeStatusFilter | null {
  if (t.status === "open") return "open";
  if (t.net_pnl == null || t.net_pnl === 0) return "wash";
  if (t.net_pnl > 0) return "win";
  return "loss";
}

export function matchesTradeStatusFilter(t: Trade, filter: TradeStatusFilter | undefined): boolean {
  if (!filter) return true;
  return tradeOutcome(t) === filter;
}

export function filterTradesByStatus(
  trades: Trade[],
  filter: TradeStatusFilter | undefined,
): Trade[] {
  if (!filter) return trades;
  return trades.filter((t) => matchesTradeStatusFilter(t, filter));
}

/** Match if the trade has any of the selected tags (OR). */
export function matchesTradeTagFilter(t: Trade, tagIds: string[] | undefined): boolean {
  if (!tagIds?.length) return true;
  const wanted = new Set(tagIds);
  return t.tags.some((tag) => wanted.has(tag.id));
}

export function filterTradesByTags(trades: Trade[], tagIds: string[] | undefined): Trade[] {
  if (!tagIds?.length) return trades;
  return trades.filter((t) => matchesTradeTagFilter(t, tagIds));
}

export type TagFacetOption = { value: string; label: string; count: number };

/** Unique tags from the current trade set, with occurrence counts. */
export function buildTagFacetOptions(trades: Trade[]): TagFacetOption[] {
  const counts = new Map<string, { label: string; count: number }>();
  for (const trade of trades) {
    for (const tag of trade.tags) {
      const cur = counts.get(tag.id);
      if (cur) cur.count += 1;
      else counts.set(tag.id, { label: tag.name, count: 1 });
    }
  }
  return [...counts.entries()]
    .map(([value, { label, count }]) => ({ value, label, count }))
    .sort((a, b) => a.label.localeCompare(b.label, undefined, { sensitivity: "base" }));
}

const MARKET_FACET_LABELS: Record<string, string> = {
  stock: "Stock",
  option: "Option",
  crypto: "Crypto",
  futures: "Futures",
  forex: "Forex",
};

export type MarketFacetOption = TagFacetOption;

/** Match if the trade's instrument type is in the selected markets (OR). */
export function matchesTradeMarketFilter(t: Trade, markets: string[] | undefined): boolean {
  if (!markets?.length) return true;
  return markets.includes(t.instrument_type);
}

export function filterTradesByMarkets(trades: Trade[], markets: string[] | undefined): Trade[] {
  if (!markets?.length) return trades;
  return trades.filter((t) => matchesTradeMarketFilter(t, markets));
}

/** Unique instrument types from the current trade set, with counts. */
export function buildMarketFacetOptions(trades: Trade[]): MarketFacetOption[] {
  const counts = new Map<string, number>();
  for (const trade of trades) {
    counts.set(trade.instrument_type, (counts.get(trade.instrument_type) ?? 0) + 1);
  }
  return [...counts.entries()]
    .map(([value, count]) => ({
      value,
      label: MARKET_FACET_LABELS[value] ?? value,
      count,
    }))
    .sort((a, b) => a.label.localeCompare(b.label, undefined, { sensitivity: "base" }));
}

export type SymbolFacetOption = TagFacetOption;

/** Unique symbols from the current trade set, with occurrence counts. */
export function buildSymbolFacetOptions(trades: Trade[]): SymbolFacetOption[] {
  const counts = new Map<string, number>();
  for (const trade of trades) {
    const symbol = trade.symbol.trim().toUpperCase();
    if (!symbol) continue;
    counts.set(symbol, (counts.get(symbol) ?? 0) + 1);
  }
  return [...counts.entries()]
    .map(([value, count]) => ({ value, label: value, count }))
    .sort((a, b) => a.label.localeCompare(b.label, undefined, { sensitivity: "base" }));
}

/** Match if the trade's symbol is in the selected list (OR). */
export function matchesTradeSymbolFilter(t: Trade, symbols: string[] | undefined): boolean {
  if (!symbols?.length) return true;
  const wanted = new Set(symbols.map((s) => s.trim().toUpperCase()).filter(Boolean));
  return wanted.has(t.symbol.trim().toUpperCase());
}

export function filterTradesBySymbols(trades: Trade[], symbols: string[] | undefined): Trade[] {
  if (!symbols?.length) return trades;
  return trades.filter((t) => matchesTradeSymbolFilter(t, symbols));
}

export interface TradeFilter {
  id: string;
  field: string;
  operator: string;
  values: string[];
}

/** Final results deliberately exclude ongoing positions, including realized partial closes. */
export function finalTradeResult(trade: Trade): "win" | "loss" | "wash" | "unknown" | null {
  if (trade.status !== "closed") return null;
  if (trade.quantities?.lifecycle === "open" || trade.quantities?.lifecycle === "partially_closed")
    return null;
  if (
    trade.quantities?.lifecycle === "unknown" ||
    trade.net_pnl == null ||
    !Number.isFinite(trade.net_pnl)
  )
    return "unknown";
  return trade.net_pnl > 0 ? "win" : trade.net_pnl < 0 ? "loss" : "wash";
}

export function holdingDurationBucket(trade: Trade): string {
  const holding = trade.holding;
  if (holding?.kind === "calendar") {
    const days = holding.calendar_days;
    if (days == null || !Number.isInteger(days) || days < 0) return "unknown";
    return days === 0
      ? "calendar_0"
      : days === 1
        ? "calendar_1"
        : days <= 5
          ? "calendar_2_5"
          : "calendar_6_plus";
  }
  if (holding && holding.kind !== "elapsed") return "unknown";
  const seconds = holding ? holding.seconds : trade.time_in_trade_secs;
  if (seconds == null || !Number.isFinite(seconds) || seconds < 0) return "unknown";
  if (seconds < 300) return "elapsed_under_5m";
  if (seconds < 1800) return "elapsed_5_30m";
  if (seconds < 3600) return "elapsed_30_60m";
  return seconds < 14400 ? "elapsed_1_4h" : "elapsed_4h_plus";
}

export function matchesTradeFilter(trade: Trade, filter: TradeFilter): boolean {
  if (!filter.values.length) return true;
  let actual: string[];
  switch (filter.field) {
    case "symbol":
      actual = [trade.symbol.trim().toUpperCase()];
      break;
    case "market":
      actual = [trade.instrument_type];
      break;
    case "tags":
      actual = trade.tags.map((tag) => tag.id);
      break;
    case "result": {
      const result = finalTradeResult(trade);
      if (result == null) return false;
      actual = [result];
      break;
    }
    case "direction": {
      const direction = trade.direction.trim().toLowerCase();
      actual = [
        isLongDirection(direction)
          ? "long"
          : direction === "short" || direction === "sell"
            ? "short"
            : "unknown",
      ];
      break;
    }
    case "progress":
      actual = [trade.quantities?.lifecycle ?? "unknown"];
      break;
    case "position_type":
      actual = [
        trade.position_type === "cash" || trade.position_type === "margin"
          ? trade.position_type
          : "unknown",
      ];
      break;
    case "duration":
      actual = [holdingDurationBucket(trade)];
      break;
    default:
      return false;
  }
  const values =
    filter.field === "symbol" ? filter.values.map((v) => v.trim().toUpperCase()) : filter.values;
  if (filter.operator === "includes_all") return values.every((v) => actual.includes(v));
  const matches = values.some((v) => actual.includes(v));
  return filter.operator === "is_not_any_of" || filter.operator === "is_not" ? !matches : matches;
}

export function applyTradeFilters(trades: Trade[], filters: TradeFilter[]): Trade[] {
  return trades.filter((trade) => filters.every((filter) => matchesTradeFilter(trade, filter)));
}

/** Preserve existing shared shortcuts while keeping Trades-only facets and operators on re-entry. */
export function tradePageFilters(state: {
  symbols?: string[];
  markets?: string[];
  tagIds?: string[];
  tradeStatus?: TradeStatusFilter;
  tradeConditions?: TradeFilter[];
}): TradeFilter[] {
  const saved = state.tradeConditions ?? [];
  const filters = saved.filter((f) => !["symbol", "market", "tags"].includes(f.field));
  for (const [field, sharedValues] of [
    ["symbol", state.symbols],
    ["market", state.markets],
    ["tags", state.tagIds],
  ] as const) {
    const savedFilter = saved.find((f) => f.field === field);
    const values = sharedValues ?? savedFilter?.values;
    if (values?.length)
      filters.push({
        id: savedFilter?.id ?? field,
        field,
        values,
        operator: savedFilter?.operator ?? "is_any_of",
      });
  }
  if (state.tradeStatus) {
    const field = state.tradeStatus === "open" ? "progress" : "result";
    const values =
      state.tradeStatus === "open" ? ["open", "partially_closed"] : [state.tradeStatus];
    return [
      ...filters.filter((f) => f.field !== "result" && f.field !== "progress"),
      { id: field, field, values, operator: "is_any_of" },
    ];
  }
  return filters;
}
