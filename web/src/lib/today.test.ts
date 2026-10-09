import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";
import { useMarketToday } from "./today";
import { useDisplayPrefs } from "./displayPrefs";

const original = useDisplayPrefs.getState().marketTimezone;
afterEach(() => {
  useDisplayPrefs.setState({ marketTimezone: original });
  vi.useRealTimers();
});
describe("market today", () => {
  it("follows Tokyo month rollover while Home remains open and switches clocks", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-31T14:59:59Z"));
    useDisplayPrefs.setState({ marketTimezone: "Asia/Tokyo" });
    const { result, unmount } = renderHook(() => useMarketToday());
    expect(result.current).toBe("2026-01-31");
    act(() => vi.advanceTimersByTime(1000));
    expect(result.current).toBe("2026-02-01");
    act(() => useDisplayPrefs.setState({ marketTimezone: "America/New_York" }));
    expect(result.current).toBe("2026-01-31");
    act(() => useDisplayPrefs.setState({ marketTimezone: "Asia/Tokyo" }));
    expect(result.current).toBe("2026-02-01");
    unmount();
  });
  it("handles New York DST without changing the day", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-08T06:59:59Z"));
    useDisplayPrefs.setState({ marketTimezone: "America/New_York" });
    const { result } = renderHook(() => useMarketToday());
    act(() => vi.advanceTimersByTime(1000));
    expect(result.current).toBe("2026-03-08");
  });
});
