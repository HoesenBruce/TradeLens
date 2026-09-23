import { render } from "@/test/render";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vite-plus/test";

vi.mock("lightweight-charts", () => {
  const series = {
    setData: vi.fn<(...args: any[]) => any>(),
    createPriceLine: vi.fn<(...args: any[]) => any>(),
  };
  const chart = {
    addSeries: vi.fn<(...args: any[]) => any>(() => series),
    applyOptions: vi.fn<(...args: any[]) => any>(),
    remove: vi.fn<(...args: any[]) => any>(),
    timeScale: () => ({ fitContent: vi.fn<(...args: any[]) => any>() }),
  };
  return {
    CandlestickSeries: {},
    ColorType: { Solid: "solid" },
    createChart: vi.fn<(...args: any[]) => any>(() => chart),
    createSeriesMarkers: vi.fn<(...args: any[]) => any>(),
  };
});

import { executionPriceLine, TradeChart } from "./TradeChart";

describe("TradeChart empty state", () => {
  it("renders an inset well with icon and message when empty", () => {
    render(
      <TradeChart
        symbol="CL1"
        bars={[]}
        fills={[]}
        interval="1"
        empty
        errorMessage="No market data for this window."
      />,
    );
    expect(screen.getByText("No market data for this window.")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: /no chart data/i })).toBeInTheDocument();
  });
});

describe("TradeChart execution prices", () => {
  it("uses the actual execution price and type", () => {
    expect(
      executionPriceLine({
        side: "buy",
        quantity: 100,
        price: 2010,
        executed_at: "2026-09-15T01:00:00Z",
        trade_type: "cash_buy",
      }),
    ).toMatchObject({ price: 2010, title: expect.stringContaining("cash_buy") });
  });
});
