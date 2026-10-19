import { t } from "@lingui/core/macro";
import type { Trade } from "./api/types";
import { intlLocale } from "./locale";

export function fmtTradeHolding(trade: Pick<Trade, "holding" | "time_in_trade_secs">) {
  const holding = trade.holding;
  let value = "—";
  let description = t({ id: "trades.holdingUnknown", message: "Holding duration is unknown." });
  if (holding?.kind === "calendar" && holding.calendar_days != null) {
    value = new Intl.NumberFormat(intlLocale(), {
      style: "unit",
      unit: "day",
      unitDisplay: "short",
    }).format(holding.calendar_days);
    const zone = holding.time_zone;
    description = t({
      id: "trades.calendarHoldingHint",
      message: `Calendar days in ${zone}; exact elapsed time is unavailable.`,
    });
  } else {
    const secs = holding ? holding.seconds : trade.time_in_trade_secs;
    if ((!holding || holding.kind === "elapsed") && secs != null && secs >= 0) {
      const minutes = Math.round(secs / 60);
      const days = Math.floor(minutes / 1440);
      const hours = Math.floor((minutes % 1440) / 60);
      const mins = minutes % 60;
      value =
        secs < 60
          ? "<1m"
          : [days ? `${days}d` : "", hours ? `${hours}h` : "", mins ? `${mins}m` : ""]
              .filter(Boolean)
              .join(" ");
      description = t({
        id: "trades.elapsedHoldingHint",
        message: "Elapsed time from precise timestamps.",
      });
    }
  }
  if (holding?.ongoing && value !== "—") {
    value = t({ id: "trades.holdingOngoing", message: `${value} · ongoing` });
  }
  return { value, description };
}
