import { describe, expect, it } from "vite-plus/test";
import type { Trade } from "./api/types";
import { applyTradeFilters } from "./tradeFilters";
import { tradesCSV } from "./tradeExport";

describe("filtered trade CSV", () => {
  it("exports all matches before pagination, preserves nulls, and escapes spreadsheet content", () => {
    const trades = Array.from(
      { length: 45 },
      (_, i) =>
        ({
          id: `id-${i}`,
          symbol: i === 0 ? '=HYPERLINK("bad")' : `S${i}`,
          direction: i < 30 ? "short" : "long",
          status: "closed",
          net_pnl: null,
          tags: [],
          pnl_currency: "USD",
        }) as unknown as Trade,
    );
    const matches = applyTradeFilters(trades, [
      { id: "direction", field: "direction", values: ["short"], operator: "is_any_of" },
    ]);
    const csv = tradesCSV(matches);
    expect(csv.split("\r\n")).toHaveLength(32);
    expect(csv).toContain('"id-29"');
    expect(csv).not.toContain('"id-30"');
    expect(csv).toContain(`"'=HYPERLINK(""bad"")"`);
    expect(csv).toContain('"unknown"');
  });
});
