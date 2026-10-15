import { afterEach, expect, it } from "vite-plus/test";
import { loadLocale } from "@/i18n";
import { fmtTradeHolding } from "./tradeHolding";
import type { Trade } from "./api/types";

const holding: NonNullable<Trade["holding"]> = {
  kind: "calendar",
  calendar_days: 0,
  seconds: null,
  time_zone: "Asia/Tokyo",
  ongoing: false,
};
afterEach(async () => {
  await loadLocale("en");
});

it("formats exact time and retains legacy server support", () => {
  expect(fmtTradeHolding({ time_in_trade_secs: 4500 }).value).toBe("1h 15m");
  expect(fmtTradeHolding({ time_in_trade_secs: 48 * 3600 + 65 * 60 }).value).toBe("2d 1h 5m");
  expect(fmtTradeHolding({ time_in_trade_secs: 0 }).value).toBe("<1m");
  expect(fmtTradeHolding({ time_in_trade_secs: null }).value).toBe("—");
});

it("formats calendar days, ongoing and unknown without claiming elapsed precision", async () => {
  expect(fmtTradeHolding({ holding, time_in_trade_secs: 86400 })).toEqual({
    value: "0 days",
    description: "Calendar days in Asia/Tokyo; exact elapsed time is unavailable.",
  });
  for (const days of [1, 3])
    expect(
      fmtTradeHolding({ holding: { ...holding, calendar_days: days }, time_in_trade_secs: null })
        .value,
    ).toBe(`${days} ${days === 1 ? "day" : "days"}`);
  expect(
    fmtTradeHolding({ holding: { ...holding, ongoing: true }, time_in_trade_secs: null }).value,
  ).toBe("0 days · ongoing");
  expect(
    fmtTradeHolding({ holding: { ...holding, kind: "unknown" }, time_in_trade_secs: 86400 }).value,
  ).toBe("—");
  await loadLocale("zh-CN");
  expect(
    fmtTradeHolding({ holding: { ...holding, calendar_days: 3 }, time_in_trade_secs: null }).value,
  ).toBe("3天");
  expect(fmtTradeHolding({ holding, time_in_trade_secs: null }).description).toContain("日历");
  await loadLocale("ja");
  expect(fmtTradeHolding({ holding, time_in_trade_secs: null }).value).toBe("0 日");
});
