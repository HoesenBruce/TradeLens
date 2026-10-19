import { useLingui } from "@lingui/react";
import type { Trade } from "@/lib/api/types";
import { fmtTradeHolding } from "@/lib/tradeHolding";

export function TradeHoldingDuration({
  trade,
}: {
  trade: Pick<Trade, "holding" | "time_in_trade_secs">;
}) {
  useLingui();
  const { value, description } = fmtTradeHolding(trade);
  return (
    <span className="tabular-nums" title={description} aria-label={`${value}. ${description}`}>
      {value}
    </span>
  );
}
