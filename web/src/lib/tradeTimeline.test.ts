import { expect, it } from "vite-plus/test";
import { fmtTradeTimeline } from "./tradeTimeline";

it("uses SBI source dates without storage clocks or holding-duration claims", () => {
  const trade = {
    opened_at: "2026-09-01T00:00:00Z",
    closed_at: "2026-09-02T00:00:00Z",
    status: "closed",
    fills: [{ details: { lot: "sbi:cash", source_time_precision: "date" } }],
  };
  expect(fmtTradeTimeline(trade, "24h")).toBe("Sep 1, 2026 → Sep 2, 2026");
  expect(fmtTradeTimeline({ ...trade, status: "open" }, "24h")).toBe("Sep 1, 2026 · still open");
  expect(fmtTradeTimeline({ ...trade, fills: [] }, "24h")).toContain("24h");
});
