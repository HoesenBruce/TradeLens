import { useLingui } from "@lingui/react";
import { t } from "@lingui/core/macro";
import type { Trade } from "@/lib/api/types";
import { quantityExplanation, quantityLifecycleLabel, quantityText, tradeQuantities } from "@/lib/tradeQuantities";
import { fmtPct } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { Pill } from "./Pill";
import { StatCell } from "./StatCell";

export function TradeQuantityCell({ trade }: { trade: Trade }) {
  useLingui();
  const q = tradeQuantities(trade);
  const open = quantityText(q.open_qty), entry = quantityText(q.entry_qty);
  const label = t({ id: "trades.openEntryDescription", message: `Open Qty ${open} / Entry Qty ${entry}` });
  const reason = quantityExplanation(q.unknown_reason);
  return <span className="whitespace-nowrap tabular-nums" aria-label={label} title={reason ? `${label}. ${reason}` : label}>{open} / {entry}</span>;
}

export function TradeQuantityDetails({ trade }: { trade: Trade }) {
  useLingui();
  const q = tradeQuantities(trade);
  const reason = quantityExplanation(q.unknown_reason);
  return <div className="flex flex-col gap-2">
    <div><Pill tone="muted">{quantityLifecycleLabel(q.lifecycle)}</Pill></div>
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
      <StatCell label={t({ id: "trades.entryQty", message: "Entry Qty" })}>{quantityText(q.entry_qty)}</StatCell>
      <StatCell label={t({ id: "trades.exitQty", message: "Exit Qty" })}>{quantityText(q.exit_qty)}</StatCell>
      <StatCell label={t({ id: "trades.openQty", message: "Open Qty" })}>{quantityText(q.open_qty)}</StatCell>
      <StatCell label={t({ id: "trades.remainingPct", message: "Remaining %" })}>{q.remaining_pct == null ? "—" : fmtPct(q.remaining_pct / 100, intlLocale(), 1)}</StatCell>
    </div>
    {reason && <p className="m-0 text-xs text-muted-foreground">{reason}</p>}
  </div>;
}
