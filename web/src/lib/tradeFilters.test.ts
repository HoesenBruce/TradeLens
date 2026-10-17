import type { Trade } from "./api/types";
import {
  buildMarketFacetOptions,
  buildSymbolFacetOptions,
  buildTagFacetOptions,
  applyTradeFilters,
  finalTradeResult,
  holdingDurationBucket,
  tradePageFilters,
  filterTradesByMarkets,
  filterTradesByStatus,
  filterTradesBySymbols,
  filterTradesByTags,
  matchesTradeMarketFilter,
  matchesTradeStatusFilter,
  matchesTradeSymbolFilter,
  matchesTradeTagFilter,
  tradeOutcome,
} from "./tradeFilters";

function partialTrade(partial: Partial<Trade> & Pick<Trade, "status" | "net_pnl">): Trade {
  return {
    id: "x",
    account_id: "a",
    symbol: "X",
    instrument_type: "stock",
    direction: "long",
    opened_at: "2026-07-01T00:00:00Z",
    closed_at: null,
    qty_opened: 1,
    qty_remaining: 0,
    avg_entry_price: 1,
    avg_exit_price: null,
    gross_pnl: null,
    fees_total: 0,
    pnl_currency: "USD",
    return_pct: null,
    time_in_trade_secs: null,
    notes: "",
    tags: [],
    ...partial,
  };
}

const closedWin = partialTrade({
  status: "closed",
  net_pnl: 100,
  tags: [{ id: "t1", user_id: "u", name: "Breakout", color: "", description: "", kind: "custom" }],
});

const closedLoss = partialTrade({
  status: "closed",
  net_pnl: -50,
  instrument_type: "option",
  tags: [{ id: "t2", user_id: "u", name: "FOMO", color: "", description: "", kind: "mistake" }],
});

const openTrade = partialTrade({
  status: "open",
  net_pnl: null,
  tags: [
    { id: "t1", user_id: "u", name: "Breakout", color: "", description: "", kind: "custom" },
    { id: "t2", user_id: "u", name: "FOMO", color: "", description: "", kind: "mistake" },
  ],
});

const washTrade = partialTrade({
  status: "closed",
  net_pnl: 0,
  tags: [],
});

describe("tradeFilters", () => {
  it("classifies trade outcomes", () => {
    expect(tradeOutcome(closedWin)).toBe("win");
    expect(tradeOutcome(closedLoss)).toBe("loss");
    expect(tradeOutcome(openTrade)).toBe("open");
    expect(tradeOutcome(washTrade)).toBe("wash");
  });

  it("filters trades by status", () => {
    const trades = [closedWin, closedLoss, openTrade, washTrade];
    expect(filterTradesByStatus(trades, "win")).toEqual([closedWin]);
    expect(filterTradesByStatus(trades, undefined)).toEqual(trades);
  });

  it("matches individual trades", () => {
    expect(matchesTradeStatusFilter(closedWin, "win")).toBe(true);
    expect(matchesTradeStatusFilter(closedWin, "loss")).toBe(false);
  });

  it("filters trades by tags (any match)", () => {
    const trades = [closedWin, closedLoss, openTrade, washTrade];
    expect(filterTradesByTags(trades, ["t1"])).toEqual([closedWin, openTrade]);
    expect(filterTradesByTags(trades, ["t1", "t2"])).toEqual([closedWin, closedLoss, openTrade]);
    expect(filterTradesByTags(trades, undefined)).toEqual(trades);
    expect(matchesTradeTagFilter(washTrade, ["t1"])).toBe(false);
  });

  it("builds tag facet options with counts", () => {
    expect(buildTagFacetOptions([closedWin, closedLoss, openTrade, washTrade])).toEqual([
      { value: "t1", label: "Breakout", count: 2 },
      { value: "t2", label: "FOMO", count: 2 },
    ]);
  });

  it("filters trades by market (any match)", () => {
    const trades = [closedWin, closedLoss, openTrade, washTrade];
    expect(filterTradesByMarkets(trades, ["option"])).toEqual([closedLoss]);
    expect(filterTradesByMarkets(trades, ["stock"])).toEqual([closedWin, openTrade, washTrade]);
    expect(filterTradesByMarkets(trades, undefined)).toEqual(trades);
    expect(matchesTradeMarketFilter(closedLoss, ["stock"])).toBe(false);
  });

  it("builds market facet options with counts", () => {
    expect(buildMarketFacetOptions([closedWin, closedLoss, openTrade, washTrade])).toEqual([
      { value: "option", label: "Option", count: 1 },
      { value: "stock", label: "Stock", count: 3 },
    ]);
  });

  it("builds symbol facet options with counts", () => {
    expect(
      buildSymbolFacetOptions([
        closedWin,
        closedLoss,
        { ...openTrade, symbol: "aapl" },
        { ...washTrade, symbol: "  " },
      ]),
    ).toEqual([
      { value: "AAPL", label: "AAPL", count: 1 },
      { value: "X", label: "X", count: 2 },
    ]);
  });

  it("filters trades by symbol (any match)", () => {
    const trades = [closedWin, { ...closedLoss, symbol: "AAPL" }, { ...openTrade, symbol: "MSFT" }];
    expect(filterTradesBySymbols(trades, ["AAPL", "MSFT"]).map((t) => t.symbol)).toEqual([
      "AAPL",
      "MSFT",
    ]);
    expect(filterTradesBySymbols(trades, undefined)).toEqual(trades);
    expect(matchesTradeSymbolFilter(closedWin, ["AAPL"])).toBe(false);
  });
});

describe("Trades page conditions", () => {
  const condition = (field: string, values: string[], operator = "is_any_of") => ({
    id: field,
    field,
    values,
    operator,
  });
  const quantities = (lifecycle: "open" | "partially_closed" | "closed" | "unknown") => ({
    entry_qty: 100,
    exit_qty: lifecycle === "closed" ? 100 : 50,
    open_qty: lifecycle === "closed" ? 0 : 50,
    remaining_pct: 50,
    lifecycle,
  });

  it("separates completed results, unknown results, and realized partial closes for both operators", () => {
    const partial = { ...closedWin, status: "open", quantities: quantities("partially_closed") };
    const unknown = {
      ...closedWin,
      net_pnl: null,
      quantities: quantities("unknown"),
      accounting_warning: "Incomplete SBI opening evidence",
    };
    const trades = [closedWin, closedLoss, washTrade, partial, unknown];
    expect(applyTradeFilters(trades, [condition("result", ["win", "wash"])])).toEqual([
      closedWin,
      washTrade,
    ]);
    expect(applyTradeFilters(trades, [condition("result", ["win"], "is_not_any_of")])).toEqual([
      closedLoss,
      washTrade,
      unknown,
    ]);
    expect(finalTradeResult(partial)).toBeNull();
    expect(finalTradeResult(unknown)).toBe("unknown");
    expect(finalTradeResult({ ...closedWin, net_pnl: Number.NaN })).toBe("unknown");
  });

  it("uses position side, not option bias, and keeps unknown categories explicit", () => {
    const put = {
      ...closedWin,
      direction: "long",
      instrument_type: "option",
      symbol: "XYZ 261218P00100000",
    };
    expect(applyTradeFilters([put], [condition("direction", ["long"])])).toEqual([put]);
    expect(applyTradeFilters([put], [condition("direction", ["short"])])).toEqual([]);
    const unknown = {
      ...closedWin,
      direction: "",
      position_type: undefined,
      quantities: undefined,
    };
    for (const field of ["direction", "position_type", "progress"]) {
      expect(applyTradeFilters([unknown], [condition(field, ["unknown"])])).toEqual([unknown]);
      expect(
        applyTradeFilters([unknown], [condition(field, ["unknown"], "is_not_any_of")]),
      ).toEqual([]);
    }
    expect(
      applyTradeFilters([{ ...put, direction: " SELL " }], [condition("direction", ["short"])]),
    ).toHaveLength(1);
  });

  it.each([
    [0, "elapsed_under_5m"],
    [299, "elapsed_under_5m"],
    [300, "elapsed_5_30m"],
    [1799, "elapsed_5_30m"],
    [1800, "elapsed_30_60m"],
    [3599, "elapsed_30_60m"],
    [3600, "elapsed_1_4h"],
    [14399, "elapsed_1_4h"],
    [14400, "elapsed_4h_plus"],
    [null, "unknown"],
    [-1, "unknown"],
    [Number.NaN, "unknown"],
    [Infinity, "unknown"],
  ])("classifies precise duration %s without display rounding", (seconds, bucket) => {
    expect(
      holdingDurationBucket({
        ...closedWin,
        holding: {
          kind: "elapsed",
          seconds,
          calendar_days: null,
          time_zone: "Asia/Tokyo",
          ongoing: false,
        },
      }),
    ).toBe(bucket);
  });

  it.each([
    [0, "calendar_0"],
    [1, "calendar_1"],
    [2, "calendar_2_5"],
    [5, "calendar_2_5"],
    [6, "calendar_6_plus"],
    [-1, "unknown"],
    [0.5, "unknown"],
    [null, "unknown"],
  ])("keeps calendar duration %s separate from precise time", (calendar_days, bucket) => {
    const trade = {
      ...closedWin,
      time_in_trade_secs: 0,
      holding: {
        kind: "calendar" as const,
        seconds: null,
        calendar_days,
        time_zone: "Asia/Tokyo",
        ongoing: true,
      },
    };
    expect(holdingDurationBucket(trade)).toBe(bucket);
    expect(applyTradeFilters([trade], [condition("duration", ["elapsed_under_5m"])])).toEqual([]);
  });

  it("does not fall back to legacy seconds when authoritative precision is unknown", () => {
    expect(
      holdingDurationBucket({
        ...closedWin,
        time_in_trade_secs: 0,
        holding: {
          kind: "unknown",
          seconds: null,
          calendar_days: null,
          time_zone: "Asia/Tokyo",
          ongoing: false,
        },
      }),
    ).toBe("unknown");
  });

  it("combines OR selections with AND dimensions across the entire input collection", () => {
    const trades = Array.from({ length: 45 }, (_, i) => ({
      ...closedWin,
      id: String(i),
      direction: i > 20 ? "short" : "long",
      quantities: quantities(i > 30 ? "partially_closed" : "open"),
    }));
    expect(
      applyTradeFilters(trades, [
        condition("direction", ["short"]),
        condition("progress", ["open", "partially_closed"]),
      ]),
    ).toHaveLength(24);
    expect(
      applyTradeFilters(trades, [
        condition("direction", ["short"]),
        condition("progress", ["closed"]),
      ]),
    ).toEqual([]);
  });

  it("applies symbol/market/tag exclusion and all-tags operators", () => {
    expect(
      applyTradeFilters(
        [closedWin, closedLoss],
        [condition("symbol", [closedWin.symbol.toLowerCase()], "is_not_any_of")],
      ),
    ).toEqual([]);
    expect(
      applyTradeFilters(
        [closedWin, closedLoss],
        [condition("market", ["option"], "is_not_any_of")],
      ),
    ).toEqual([closedWin]);
    expect(
      applyTradeFilters([closedWin, closedLoss], [condition("tags", ["t1"], "is_not_any_of")]),
    ).toEqual([closedLoss]);
    expect(
      applyTradeFilters([closedWin, closedLoss], [condition("tags", ["t1", "t2"], "includes_all")]),
    ).toEqual([]);
  });

  it("preserves operators on re-entry and maps legacy open shortcuts to both ongoing stages", () => {
    const saved = [condition("symbol", ["X"], "is_not_any_of"), condition("direction", ["short"])];
    expect(tradePageFilters({ symbols: ["X"], tradeConditions: saved })).toEqual([
      saved[1],
      saved[0],
    ]);
    expect(tradePageFilters({ tradeStatus: "open" })).toEqual([
      condition("progress", ["open", "partially_closed"]),
    ]);
    expect(tradePageFilters({ tradeStatus: "loss" })).toEqual([condition("result", ["loss"])]);
  });
});

it("keeps UI filter IDs stable for multi-selection in shared facets", () => {
  const filter = { id: "generated-id", field: "symbol", values: ["AAPL"], operator: "is_any_of" };
  expect(tradePageFilters({ symbols: ["AAPL"], tradeConditions: [filter] })).toEqual([filter]);
  const exclusion = { ...filter, operator: "is_not_any_of" };
  expect(tradePageFilters({ tradeConditions: [exclusion] })).toEqual([exclusion]);
});

it("keeps known final P&L when an SBI warning only describes date precision or estimated basis", () => {
  expect(
    finalTradeResult({
      ...closedWin,
      accounting_warning: "SBI date-level reconciliation: exact elapsed holding time is unknown",
    }),
  ).toBe("win");
  expect(
    finalTradeResult({
      ...closedWin,
      accounting_warning: "SBI cash P&L uses an average-cost estimate",
    }),
  ).toBe("win");
});
