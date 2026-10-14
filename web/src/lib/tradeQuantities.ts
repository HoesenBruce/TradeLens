import type { Trade, TradeQuantities } from "./api/types";
import { t } from "@lingui/core/macro";
import { fmtNumber } from "./format";

const unavailable: TradeQuantities = {
  entry_qty: null,
  exit_qty: null,
  open_qty: null,
  remaining_pct: null,
  lifecycle: "unknown",
  unknown_reason: "metadata_unavailable",
};

export function tradeQuantities(trade: Pick<Trade, "quantities">): TradeQuantities {
  return trade.quantities ?? unavailable;
}

export function quantityText(value: number | null): string {
  return value == null ? "—" : fmtNumber(value);
}

export function quantityExplanation(reason?: string): string | undefined {
  switch (reason) {
    case "incomplete_opening":
      return t({
        id: "trades.quantityMissingOpening",
        message: "Quantity is unknown because opening history is incomplete.",
      });
    case "unverified_share_units":
      return t({
        id: "trades.quantityShareUnits",
        message:
          "Quantity is unknown because share units across a corporate action are unverified.",
      });
    case "inconsistent_inventory":
      return t({
        id: "trades.quantityInconsistent",
        message: "Quantity is unknown because the stored trade inventory is inconsistent.",
      });
    case "metadata_unavailable":
      return t({
        id: "trades.quantityServerUnavailable",
        message: "Quantity details are unavailable from this server.",
      });
    default:
      return undefined;
  }
}

export function quantityLifecycleLabel(lifecycle: TradeQuantities["lifecycle"]): string {
  switch (lifecycle) {
    case "open":
      return t({ id: "trades.lifecycleOpen", message: "Position open" });
    case "partially_closed":
      return t({ id: "trades.lifecyclePartial", message: "Partially closed" });
    case "closed":
      return t({ id: "trades.lifecycleClosed", message: "Position closed" });
    case "unknown":
      return t({ id: "trades.lifecycleUnknown", message: "Position unknown" });
  }
}
