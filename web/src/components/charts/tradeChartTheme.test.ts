import { describe, expect, it } from "vite-plus/test";
import { candleColors, tradeChartTheme } from "./tradeChartTheme";

describe("candleColors", () => {
  it("maps both conventions without changing buy and sell markers", () => {
    expect(candleColors(false)).toEqual({ up: tradeChartTheme.up, down: tradeChartTheme.down });
    expect(candleColors(true)).toEqual({ up: tradeChartTheme.down, down: tradeChartTheme.up });
    expect(tradeChartTheme.buyMarker).toBe("#4fa5ff");
  });
});
