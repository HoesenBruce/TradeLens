import { MetricHelp } from "./MetricHelp";
import { useLingui } from "@lingui/react";
import { t } from "@lingui/core/macro";
import type { Trade, TradeQuantities } from "@/lib/api/types";
import {
  quantityClosedPct,
  quantityExplanation,
  quantityLifecycleLabel,
  quantityText,
  tradeQuantities,
} from "@/lib/tradeQuantities";
import { fmtPct } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { Pill } from "./Pill";
import { cn } from "@/lib/cn";

function progressPercent(percent: number): string {
  if (percent > 0 && percent < 0.1) return `<${fmtPct(0.001, intlLocale(), 1)}`;
  if (percent > 99.9 && percent < 100) return `>${fmtPct(0.999, intlLocale(), 1)}`;
  return fmtPct(percent / 100, intlLocale(), 1);
}

function quantitySummary(q: TradeQuantities) {
  const percent = quantityClosedPct(q);
  const closed = quantityText(q.exit_qty),
    entry = quantityText(q.entry_qty),
    remaining = quantityText(q.open_qty);
  const remainingPct = percent == null ? "—" : progressPercent(100 - percent);
  const pct = percent == null ? "—" : progressPercent(percent);
  const closedDescription = t({
    id: "trades.closedQuantitySummary",
    message: `Closed ${closed} / ${entry}`,
  });
  const remainingDescription = t({
    id: "trades.remainingQuantitySummary",
    message: `Remaining ${remaining} (${remainingPct})`,
  });
  const summary = t({ id: "trades.closedPercent", message: `${pct} closed` });
  const lifecycle = quantityLifecycleLabel(q.lifecycle);
  return { percent, remainingPct, closedDescription, remainingDescription, summary, lifecycle };
}

function QuantityProgress({
  percent,
  description,
  lifecycle,
}: {
  percent: number | null;
  description: string;
  lifecycle: TradeQuantities["lifecycle"];
}) {
  return (
    <div
      className="h-2 w-full overflow-hidden rounded-full bg-muted"
      role={percent == null ? undefined : "progressbar"}
      aria-label={
        percent == null ? undefined : t({ id: "trades.closeProgress", message: "Close progress" })
      }
      aria-valuemin={percent == null ? undefined : 0}
      aria-valuemax={percent == null ? undefined : 100}
      aria-valuenow={percent ?? undefined}
      aria-valuetext={percent == null ? undefined : description}
      aria-hidden={percent == null ? true : undefined}
    >
      <div
        className={cn("h-full rounded-full", lifecycle === "closed" ? "bg-success" : "bg-primary")}
        style={{ width: `${percent ?? 0}%` }}
      />
    </div>
  );
}

export function TradeQuantityCell({ trade }: { trade: Pick<Trade, "quantities"> }) {
  useLingui();
  const q = tradeQuantities(trade);
  const view = quantitySummary(q);
  const reason = quantityExplanation(q.unknown_reason);
  return (
    <div
      className="flex min-w-0 flex-col gap-1 whitespace-normal text-left text-xs"
      data-slot="trade-quantity"
    >
      <span className="text-[10px] font-medium text-muted-foreground">{view.lifecycle}</span>
      <QuantityProgress
        percent={view.percent}
        lifecycle={q.lifecycle}
        description={`${view.lifecycle}. ${view.summary}. ${view.closedDescription}. ${view.remainingDescription}`}
      />
      <span className="wrap-anywhere tabular-nums" data-slot="quantity-value">
        {view.closedDescription} · {view.summary}
      </span>
      <span className="wrap-anywhere tabular-nums text-muted-foreground" data-slot="quantity-value">
        {view.remainingDescription}
      </span>
      {reason && <span className="wrap-anywhere text-muted-foreground">{reason}</span>}
    </div>
  );
}

export function TradeQuantityDetails({ trade }: { trade: Pick<Trade, "quantities"> }) {
  useLingui();
  const q = tradeQuantities(trade);
  const view = quantitySummary(q);
  const reason = quantityExplanation(q.unknown_reason);
  const metrics: [string, number | null][] = [
    [t({ id: "trades.entryQty", message: "Entry Qty" }), q.entry_qty],
    [t({ id: "trades.exitQty", message: "Exit Qty" }), q.exit_qty],
    [t({ id: "trades.openQty", message: "Open Qty" }), q.open_qty],
  ];
  return (
    <div
      className="@container/quantities flex min-w-0 w-full flex-col gap-3 rounded-lg border border-border p-3"
      data-slot="trade-quantity"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Pill
          tone={
            q.lifecycle === "closed"
              ? "pos"
              : q.lifecycle === "partially_closed"
                ? "amber"
                : "muted"
          }
        >
          {view.lifecycle}
        </Pill>
        <MetricHelp metric="quantities" />
        {view.percent != null && (
          <span className="text-xs font-medium tabular-nums">{view.summary}</span>
        )}
      </div>
      <QuantityProgress
        percent={view.percent}
        lifecycle={q.lifecycle}
        description={`${view.lifecycle}. ${view.summary}. ${view.closedDescription}. ${view.remainingDescription}`}
      />
      <dl className="m-0 grid grid-cols-1 gap-3 @min-[40rem]/quantities:grid-cols-3">
        {metrics.map(([label, value]) => (
          <div className="min-w-0" key={label}>
            <dt className="text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
              {label}
            </dt>
            <dd
              className="m-0 mt-1 text-base font-semibold whitespace-normal wrap-anywhere tabular-nums"
              data-slot="quantity-value"
            >
              {quantityText(value)}
            </dd>
          </div>
        ))}
      </dl>
      <p className="m-0 text-xs wrap-anywhere text-muted-foreground">
        <span>{t({ id: "trades.remainingPct", message: "Remaining %" })}</span>:{" "}
        <span className="tabular-nums">{view.remainingPct}</span>
      </p>
      {reason && <p className="m-0 text-xs text-muted-foreground">{reason}</p>}
    </div>
  );
}
