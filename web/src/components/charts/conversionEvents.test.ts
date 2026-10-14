import { expect, it } from "vite-plus/test";
import { conversionEvents } from "./conversionEvents";
import { fillMarkers } from "./TradeChart";
const margin = {
  side: "sell",
  quantity: 11000,
  price: 3159,
  executed_at: "2026-08-04T00:00:00Z",
  details: {
    event_type: "position_conversion",
    conversion_type: "genbiki",
    conversion_id: "c1",
    lot: "sbi:margin-long",
  },
};
const cash = { ...margin, side: "buy", details: { ...margin.details, lot: "sbi:cash" } };
it("keeps date-only SBI markers on daily bars and omits fabricated intraday placement", () => {
  const date = {
    ...cash,
    executed_at: "2026-09-01T00:00:00.000002Z",
    details: { lot: "sbi:cash", source_time_precision: "date" },
  };
  expect(fillMarkers([date], "Conversion", "en-US", "Asia/Tokyo", "1")).toEqual([]);
  expect(fillMarkers([date], "Conversion", "en-US", "Asia/Tokyo", "D")[0]?.time).toBe(
    Date.parse("2026-09-01T00:00:00Z") / 1000,
  );
  expect(
    fillMarkers([{ ...date, details: null }], "Conversion", "en-US", "Asia/Tokyo", "1"),
  ).toHaveLength(1);
});
it("groups explicit linked conversions, preserves ordinary fills and distinct IDs", () => {
  const ordinary = { ...cash, details: null };
  const other = { ...cash, details: { ...cash.details, conversion_id: "c2" } };
  expect(conversionEvents([margin, cash, ordinary, other]).map((e) => e.conversion)).toEqual([
    true,
    false,
    true,
  ]);
  expect(conversionEvents([ordinary, { ...ordinary, side: "sell" }])).toHaveLength(2);
  for (const field of ["event_type", "conversion_type", "conversion_id", "lot"])
    expect(
      conversionEvents([{ ...margin, details: { ...margin.details, [field]: "" } }])[0]!.conversion,
    ).toBe(false);
  expect(conversionEvents([margin, { ...cash, quantity: 10 }]).every((e) => !e.conversion)).toBe(
    true,
  );
  expect(conversionEvents([margin])[0]!.conversion).toBe(true);
  for (const label of ["Margin → Cash", "現引", "信用转现物"])
    expect(fillMarkers([margin, cash], label, "en-US")).toMatchObject([
      { shape: "circle", position: "aboveBar", text: `◆ ${label} 11,000` },
    ]);
});
