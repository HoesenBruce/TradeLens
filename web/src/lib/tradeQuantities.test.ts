import { expect, it } from "vite-plus/test";
import { quantityClosedPct, quantityText } from "./tradeQuantities";
import type { TradeQuantities } from "./api/types";

const partial: TradeQuantities = {
  entry_qty: 500,
  exit_qty: 400,
  open_qty: 100,
  remaining_pct: 20,
  lifecycle: "partially_closed",
};
it("bounds segment progress and guards unknown/zero denominators", () => {
  expect(quantityClosedPct(partial)).toBe(80);
  expect(quantityClosedPct({ ...partial, exit_qty: 700 })).toBe(100);
  expect(quantityClosedPct({ ...partial, exit_qty: -1 })).toBe(0);
  expect(quantityClosedPct({ ...partial, entry_qty: 0 })).toBeNull();
  expect(quantityClosedPct({ ...partial, entry_qty: Infinity })).toBeNull();
  expect(quantityClosedPct({ ...partial, exit_qty: NaN })).toBeNull();
  expect(quantityClosedPct({ ...partial, lifecycle: "unknown" })).toBeNull();
});
it("retains supported fractional units and full integers without floating subtraction noise", () => {
  expect(quantityText(0.123456789)).toBe("0.123456789");
  expect(quantityText(0.3 - 0.1)).toBe("0.2");
  expect(quantityText(1234567890123.125)).toBe("1,234,567,890,123.125");
  expect(quantityText(1e-9)).toBe("0.000000001");
  expect(quantityText(null)).toBe("—");
});
