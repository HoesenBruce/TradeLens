import { useLingui } from "@lingui/react/macro";
import type { Trade } from "@/lib/api/types";
import { Pill } from "./Pill";

export function TradePositionTypeBadge({ trade }: { trade: Pick<Trade, "position_type"> }) {
  const { t } = useLingui();
  if (!trade.position_type) return null;
  const label =
    trade.position_type === "cash"
      ? t({ id: "trades.positionCash", message: "Cash" })
      : trade.position_type === "margin"
        ? t({ id: "trades.positionMargin", message: "Margin" })
        : t({ id: "trades.positionTypeUnknown", message: "Type unknown" });
  return <Pill tone="muted">{label}</Pill>;
}
