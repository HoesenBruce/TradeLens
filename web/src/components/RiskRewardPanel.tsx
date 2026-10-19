import type { MetricId } from "@/lib/metrics/definitions";
import { MetricLabel } from "./MetricHelp";
import type { TradeDetail } from "@/lib/api/types";

import type { ReactNode } from "react";
import { Money } from "./ResponsiveValue";
import { computeRiskReward, type RiskRewardMetrics } from "@/lib/riskReward";
import { cn } from "@/lib/cn";
import { cardSectionLabelClass } from "./StatCell";
import { pnlColor } from "./theme-tokens";

const sectionLabelClass = cn(cardSectionLabelClass, "mb-3");

function StatCell({
  label,
  metric,
  value,
  valueClassName,
}: {
  label: string;
  metric?: MetricId;
  value: ReactNode;
  valueClassName?: string;
}) {
  return (
    <div className="flex min-w-0 max-w-full flex-col gap-0.5 wrap-anywhere">
      <span className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
        <MetricLabel metric={metric}>{label}</MetricLabel>
      </span>
      <span className={cn("text-sm tabular-nums text-foreground", valueClassName)}>{value}</span>
    </div>
  );
}

export interface RiskRewardPanelProps {
  trade: TradeDetail;
  metrics?: RiskRewardMetrics;
  className?: string;
  /** When true, omit the panel entirely if every metric is empty. */
  hideWhenEmpty?: boolean;
}

export function RiskRewardPanel({
  trade,
  metrics,
  className,
  hideWhenEmpty = false,
}: RiskRewardPanelProps) {
  const m = metrics ?? computeRiskReward(trade);
  const currency = trade.pnl_currency;

  const maxProfitValue =
    m.maxProfit != null ? (
      <Money value={m.maxProfit} currency={currency} />
    ) : m.target != null ? (
      "∞"
    ) : null;

  const cells: { label: string; value: ReactNode; valueClassName?: string; metric?: MetricId }[] =
    [];

  if (m.target != null) {
    cells.push({ label: "Target", value: <Money value={m.target} currency={currency} /> });
  }
  if (m.stop != null) {
    cells.push({ label: "Stop", value: <Money value={m.stop} currency={currency} /> });
  }
  if (maxProfitValue != null) {
    cells.push({
      label: "Max profit",
      value: maxProfitValue,
      valueClassName: m.maxProfit != null ? pnlColor(m.maxProfit) : undefined,
    });
  }
  if (m.maxLoss != null) {
    cells.push({
      label: "Max loss",
      value: <Money value={m.maxLoss} currency={currency} />,
      valueClassName: pnlColor(m.maxLoss),
    });
  }
  if (m.breakeven != null) {
    cells.push({
      label: "Breakeven",
      value: <Money value={m.breakeven} currency={currency} />,
    });
  }
  if (m.plannedRR != null) {
    cells.push({ metric: "plannedRR", label: "Planned R:R", value: `${m.plannedRR.toFixed(2)}:1` });
  }
  if (trade.r_multiple != null) {
    cells.push({
      metric: "realizedR",
      label: "R-multiple",
      value: `${trade.r_multiple >= 0 ? "+" : ""}${trade.r_multiple.toFixed(2)}R`,
      valueClassName: pnlColor(trade.r_multiple),
    });
  }

  if (cells.length === 0) {
    if (hideWhenEmpty) return null;
    return (
      <section className={cn("px-4 py-3", className)}>
        <p className={sectionLabelClass}>Risk / Reward</p>
        <p className="text-xs text-muted-foreground">
          No plan set — add target and stop on the full page.
        </p>
      </section>
    );
  }

  return (
    <section className={cn("px-4 py-3", className)}>
      <p className={sectionLabelClass}>Risk / Reward</p>
      <div className="grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-3 lg:grid-cols-4">
        {cells.map((cell) => (
          <StatCell
            key={cell.label}
            metric={cell.metric}
            label={cell.label}
            value={cell.value}
            valueClassName={cell.valueClassName}
          />
        ))}
      </div>
    </section>
  );
}
