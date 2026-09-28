import { describe, expect, it } from "vite-plus/test";
import { combineAccountValues } from "./accountValue";
import type { AccountValue } from "./api/types";

function series(currency: string, capital: number, value = capital): AccountValue {
  return {
    currency,
    timezone: "Asia/Tokyo",
    adjustment_status: "unadjusted",
    points: [
      {
        date: "2025-05-19",
        estimated_account_value: value,
        contributed_capital: capital,
        cash_balance: capital,
        open_position_value: value - capital,
        realized_pnl: 0,
        unrealized_pnl: value - capital,
        status: "complete",
        warnings: [],
      },
    ],
  };
}

describe("historical account value currency", () => {
  it("leaves JPY → JPY unchanged even if a USD rate exists", () => {
    const result = combineAccountValues([series("JPY", 1_350_000, 1_500_000)], "JPY", { USD: 157 });
    expect(result.currency).toBe("JPY");
    expect(result.points[0].contributed_capital).toBe(1_350_000);
    expect(result.points[0].estimated_account_value).toBe(1_500_000);
  });

  it("converts USD exactly once and then adds JPY on the same basis", () => {
    const result = combineAccountValues(
      [series("JPY", 1_350_000, 1_500_000), series("USD", 10_000, 11_000)],
      "JPY",
      { USD: 157 },
    );
    expect(result.points[0]).toMatchObject({
      contributed_capital: 2_920_000,
      estimated_account_value: 3_227_000,
      cash_balance: 2_920_000,
      open_position_value: 307_000,
      unrealized_pnl: 307_000,
    });
    expect(result.points[0].estimated_account_value! - result.points[0].contributed_capital).toBe(
      307_000,
    );
    const usd = combineAccountValues([series("USD", 10_000)], "JPY", { USD: 157 });
    expect(usd.points[0].estimated_account_value).toBe(1_570_000);
  });

  it("converts in the reverse direction and preserves same-currency aggregation", () => {
    expect(
      combineAccountValues([series("JPY", 1_350_000), series("USD", 10_000)], "USD", {
        JPY: 1 / 157,
      }).points[0].contributed_capital,
    ).toBeCloseTo(18_598.72611465);
    expect(
      combineAccountValues([series("JPY", 1_350_000), series("JPY", 650_000)], "JPY", {}).points[0]
        .contributed_capital,
    ).toBe(2_000_000);
  });

  it("preserves unavailable values and warnings without mutating source data", () => {
    const missing = series("USD", 10_000);
    Object.assign(missing.points[0], {
      estimated_account_value: null,
      open_position_value: null,
      unrealized_pnl: null,
      status: "incomplete_missing_price",
      warnings: [{ code: "missing_price", date: "2025-05-19", message: "missing" }],
    });
    const original = structuredClone(missing);
    const result = combineAccountValues([series("JPY", 1_350_000), missing], "JPY", { USD: 157 });
    expect(result.points[0]).toMatchObject({
      contributed_capital: 2_920_000,
      estimated_account_value: null,
      open_position_value: null,
      unrealized_pnl: null,
      status: "incomplete_missing_price",
      warnings: missing.points[0].warnings,
    });
    expect(missing).toEqual(original);
  });

  it("aligns dates, treats pre-inception dates as zero, and leaves gaps unavailable", () => {
    const jpy = series("JPY", 1_350_000);
    jpy.points.push(
      { ...jpy.points[0], date: "2025-05-20" },
      { ...jpy.points[0], date: "2025-05-21" },
    );
    const usd = series("USD", 10_000);
    usd.points[0].date = "2025-05-20";
    expect(
      combineAccountValues([jpy, usd], "JPY", { USD: 157 }).points.map(
        (p) => p.estimated_account_value,
      ),
    ).toEqual([1_350_000, 2_920_000, null]);
  });

  it.each([undefined, 0, -1, Infinity, NaN])("rejects unusable FX rates (%s)", (rate) => {
    expect(() => combineAccountValues([series("USD", 10_000)], "JPY", { USD: rate })).toThrow(
      "Missing FX rate",
    );
  });
});
