import { render } from "@/test/render";

import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import type { Execution } from "@/lib/api/types";
import type { MarketBar } from "@/lib/api/market";

const createPriceLine = vi.fn<(...args: any[]) => any>();
const setData = vi.fn<(...args: any[]) => any>();
const setMarkers = vi.fn<(...args: any[]) => any>();
const createSeriesMarkers = vi.fn<(...args: any[]) => any>(() => ({ setMarkers }));
const setVisibleRange = vi.fn<(...args: any[]) => any>();

vi.mock("lightweight-charts", () => {
  const series = {
    setData: (...args: any[]) => setData(...args),
    applyOptions: vi.fn<(...args: any[]) => any>(),
    createPriceLine: (...args: any[]) => createPriceLine(...args),
    priceLines: () => [],
    removePriceLine: vi.fn<(...args: any[]) => any>(),
  };
  const chart = {
    addSeries: vi.fn<(...args: any[]) => any>(() => series),
    applyOptions: vi.fn<(...args: any[]) => any>(),
    remove: vi.fn<(...args: any[]) => any>(),
    timeScale: () => ({
      fitContent: vi.fn<(...args: any[]) => any>(),
      setVisibleRange: (...args: any[]) => setVisibleRange(...args),
    }),
  };
  return {
    CandlestickSeries: {},
    ColorType: { Solid: "solid" },
    createChart: vi.fn<(...args: any[]) => any>(() => chart),
    createSeriesMarkers: (...args: any[]) => createSeriesMarkers(...args),
  };
});

import { TradeChart } from "./TradeChart";

const T0 = 1_752_600_000;

function bar(time: number, close: number): MarketBar {
  return { time, open: close, high: close, low: close, close, volume: 0 };
}

const bars = [bar(T0, 10), bar(T0 + 60, 11), bar(T0 + 120, 12)];

const fills: Execution[] = [
  {
    id: "e1",
    user_id: "u1",
    account_id: "a1",
    external_id: null,
    symbol: "AAPL",
    instrument_type: "stock",
    side: "buy",
    quantity: 100,
    price: 11,
    fees: 0,
    commission: 0,
    executed_at: new Date((T0 + 60) * 1000).toISOString(),
    multiplier: 1,
    details: null,
    import_batch_id: null,
    dedup_hash: "h",
    created_at: new Date((T0 + 60) * 1000).toISOString(),
  },
];

describe("TradeChart replay mode", () => {
  beforeEach(() => {
    createPriceLine.mockClear();
    setData.mockClear();
    setMarkers.mockClear();
    createSeriesMarkers.mockClear();
    setVisibleRange.mockClear();
  });

  it("hides bars past the cursor (transparent, keeping the time axis) and their fill markers", () => {
    render(
      <TradeChart symbol="AAPL" bars={bars} fills={fills} interval="1" replayUpTo={T0 + 60} />,
    );
    const points = setData.mock.calls.at(-1)![0];
    // All bars stay in the series — trailing whitespace would be dropped by
    // lightweight-charts and collapse the axis — but future ones are painted
    // transparent.
    expect(points).toHaveLength(3);
    expect(points[0]).toHaveProperty("open");
    expect(points[0]).not.toHaveProperty("color");
    expect(points[1].color).toBe("rgba(0,0,0,0)");
    expect(points[2].wickColor).toBe("rgba(0,0,0,0)");
    // The only fill sits in the second bar, which the cursor hasn't reached.
    expect(createSeriesMarkers).not.toHaveBeenCalled();
  });

  it("reveals candles and markers once the cursor passes them", () => {
    render(
      <TradeChart symbol="AAPL" bars={bars} fills={fills} interval="1" replayUpTo={T0 + 180} />,
    );
    const points = setData.mock.calls.at(-1)![0];
    expect(points[2]).not.toHaveProperty("color");
    expect(createSeriesMarkers).toHaveBeenCalledTimes(1);
    expect(createSeriesMarkers.mock.calls[0]![1]).toHaveLength(1);
  });

  it("shows all candles untinted when replay is off", () => {
    render(<TradeChart symbol="AAPL" bars={bars} fills={[]} interval="1" />);
    const points = setData.mock.calls.at(-1)![0];
    expect(points.every((p: object) => "open" in p && !("color" in p))).toBe(true);
  });

  it("keeps the requested trade window visible while older bars remain pannable", () => {
    const from = new Date(T0 * 1000).toISOString();
    const to = new Date((T0 + 120) * 1000).toISOString();
    render(
      <TradeChart
        symbol="AAPL"
        bars={bars}
        fills={[]}
        interval="D"
        visibleFrom={from}
        visibleTo={to}
      />,
    );
    expect(setVisibleRange).toHaveBeenCalledTimes(1);
  });

  it("renders the replay toggle with a state-dependent label", () => {
    const { getByLabelText, rerender } = render(
      <TradeChart symbol="AAPL" bars={bars} fills={[]} interval="1" onToggleReplay={() => {}} />,
    );
    expect(getByLabelText("Replay trade")).toBeInTheDocument();
    rerender(
      <TradeChart
        symbol="AAPL"
        bars={bars}
        fills={[]}
        interval="1"
        onToggleReplay={() => {}}
        replayActive
        replayUpTo={T0 + 60}
      />,
    );
    expect(getByLabelText("Exit replay")).toBeInTheDocument();
  });
});

it("renders localized conversion pairs without execution price lines in embedded and expanded charts", async () => {
  const { loadLocale } = await import("@/i18n");
  const details = {
    event_type: "position_conversion",
    conversion_type: "genbiki",
    conversion_id: "c1",
    lot: "sbi:margin-long",
  };
  const pair = [
    { ...fills[0]!, side: "sell", details },
    { ...fills[0]!, details: { ...details, lot: "sbi:cash" } },
  ];
  for (const [locale, label] of [
    ["en", "Margin → Cash"],
    ["ja", "現引"],
    ["zh-CN", "信用转现物"],
  ]) {
    await loadLocale(locale!);
    for (const expanded of [false, true]) {
      createPriceLine.mockClear();
      const { unmount } = render(
        <TradeChart
          symbol="1515"
          bars={bars}
          fills={pair}
          interval="1"
          hideHeaderLabel={expanded}
          height={expanded ? 480 : 280}
        />,
      );
      expect(createSeriesMarkers.mock.calls.at(-1)![1]).toMatchObject([
        { shape: "circle", text: expect.stringContaining(label!) },
      ]);
      expect(createPriceLine).not.toHaveBeenCalled();
      unmount();
    }
  }
  await loadLocale("en");
});
