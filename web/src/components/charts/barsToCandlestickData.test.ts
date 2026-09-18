import { describe, expect, it } from "vite-plus/test";
import { barsToCandlestickData } from "./barsToCandlestickData";

function bar(time: number, close = 100) {
  return { time, open: close, high: close + 1, low: close - 1, close, volume: 1 };
}

describe("barsToCandlestickData", () => {
  it("passes through contiguous bars unchanged", () => {
    const bars = [bar(1_000), bar(1_300), bar(1_600)];
    const points = barsToCandlestickData(bars, "5", "UTC");
    expect(points).toHaveLength(3);
    expect(points.every((p) => "open" in p)).toBe(true);
  });

  it("keeps TSE lunch, overnight, and weekend gaps compact", () => {
    const times = [
      "2024-07-12T06:00:00.000Z", // Friday 15:00 JST
      "2024-07-16T00:00:00.000Z", // Tuesday 09:00 JST (Monday holiday)
      "2024-07-16T02:30:00.000Z", // 11:30 JST
      "2024-07-16T03:30:00.000Z", // 12:30 JST
      "2024-07-17T00:00:00.000Z", // next session
    ].map((value) => Math.floor(Date.parse(value) / 1000));
    const points = barsToCandlestickData(
      times.map((time) => bar(time)),
      "60",
      "Asia/Tokyo",
    );

    expect(points).toHaveLength(times.length);
    expect(points.every((point) => "open" in point)).toBe(true);
    expect(points.map((point) => point.time)).toEqual(times.map((time) => time + 9 * 60 * 60));
  });

  it("shifts labels to America/New_York wall clock", () => {
    const utc = Math.floor(Date.parse("2024-07-15T13:30:00.000Z") / 1000);
    const points = barsToCandlestickData([bar(utc)], "5", "America/New_York");
    expect(points[0]).toMatchObject({
      time: Math.floor(Date.UTC(2024, 6, 15, 9, 30, 0) / 1000),
      open: 100,
    });
  });

  it("keeps daily bars compact across market holidays", () => {
    const friday = Math.floor(Date.parse("2024-12-27T00:00:00.000Z") / 1000);
    const monday = Math.floor(Date.parse("2025-01-06T00:00:00.000Z") / 1000);
    const points = barsToCandlestickData([bar(friday), bar(monday)], "D", "UTC");

    expect(points.map((point) => point.time)).toEqual([friday, monday]);
  });
});
