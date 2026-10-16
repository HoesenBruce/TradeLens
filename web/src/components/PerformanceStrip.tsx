import { MetricHelp } from "@/components/MetricHelp";
import type { MetricId } from "@/lib/metrics/definitions";
import { useLingui as useLinguiMacro } from "@lingui/react/macro";
import { StatBar } from "./StatBar";
import { ResponsiveValue } from "./ResponsiveValue";
import { pnlColor } from "./theme-tokens";
import type { Summary, Trade } from "@/lib/api/types";
import { cn } from "@/lib/cn";

import { fmtPct } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { useMoneyFormatters } from "@/lib/useMoneyFormatters";
import type { TradeStatusFilter } from "@/lib/tradeFilters";

export interface PerformanceStripProps {
  summary: Summary;
  trades: Trade[];
  currency: string;
  /** Multiply money fields by this FX rate (base → display). */
  fxRate?: number;
  tradeStatusFilter?: TradeStatusFilter;
  onToggleTradeStatus?: (filter: TradeStatusFilter) => void;
}

function Meta({
  label,
  metric,
  value,
  compactValue,
  className,
}: {
  label: string;
  metric?: MetricId;
  value: string;
  compactValue?: string;
  className?: string;
}) {
  return (
    <p className="m-0 flex min-w-0 max-w-full flex-wrap items-baseline gap-1.5 text-[13px] tabular-nums">
      <span className="min-w-0 max-w-full wrap-anywhere font-medium tracking-wide text-muted-foreground">
        {label}
        {metric && <MetricHelp metric={metric} />}
      </span>
      {compactValue ? (
        <ResponsiveValue
          value={value}
          compactValue={compactValue}
          className={cn("font-medium", className)}
        />
      ) : (
        <span className={cn("font-medium", className)}>{value}</span>
      )}
    </p>
  );
}

export function PerformanceStrip({
  summary,
  trades,
  currency,
  fxRate = 1,
  tradeStatusFilter,
  onToggleTradeStatus,
}: PerformanceStripProps) {
  const { t: tr } = useLinguiMacro();

  const { fmtMoney, fmtSignedMoney, fmtMoneyCompact, fmtSignedMoneyCompact } = useMoneyFormatters();
  const locale = intlLocale();
  const total = Math.max(summary.total_trades, 1);
  const allTotal = Math.max(trades.length, 1);
  const openCount = trades.filter((t) => t.status === "open").length;
  const toggle = (f: TradeStatusFilter) => onToggleTradeStatus?.(f);
  // gross_profit/gross_loss are net-based win/loss buckets, so their difference is
  // just net_pnl. Real gross P&L is net plus the fees that were deducted from it.
  const gross = summary.net_pnl + summary.total_fees;
  const money = (v: number) => v * fxRate;

  return (
    <div data-testid="performance-strip" className="flex h-full min-w-0 flex-col gap-3">
      {/* Net hero */}
      <section className="flex min-w-0 flex-1 flex-col rounded-lg bg-card p-5">
        <p className="self-start text-[12px] font-semibold tracking-wide text-chart-3">
          {tr({ id: "accounts.performance", message: "Performance" })}
        </p>
        <p className="mt-3 self-start text-[12px] font-medium tracking-wide text-muted-foreground">
          {tr({ id: "accounts.net", message: "Net" })}
          <MetricHelp metric="pnl" />
        </p>
        <div className="mt-3 flex flex-1 flex-col items-center justify-center text-center">
          <ResponsiveValue
            value={fmtSignedMoney(money(summary.net_pnl), currency, locale)}
            compactValue={fmtSignedMoneyCompact(money(summary.net_pnl), currency, locale)}
            className={cn(
              "text-[32px] font-semibold leading-none tracking-[-0.04em] tabular-nums",
              pnlColor(summary.net_pnl),
              summary.net_pnl > 0 && "drop-shadow-[0_0_28px_var(--profit)]",
              summary.net_pnl < 0 && "drop-shadow-[0_0_28px_var(--loss)]",
            )}
          />
          <div className="mt-4 grid w-full min-w-0 grid-cols-[repeat(auto-fit,minmax(min(100%,140px),1fr))] gap-x-4 gap-y-1.5">
            <Meta
              metric="pnl"
              label={tr({ id: "accounts.gross", message: "Gross" })}
              value={fmtSignedMoney(money(gross), currency, locale)}
              compactValue={fmtSignedMoneyCompact(money(gross), currency, locale)}
              className={pnlColor(gross)}
            />
            <Meta
              metric="fees"
              label={tr({ id: "accounts.fees", message: "Fees" })}
              value={fmtMoney(money(summary.total_fees), currency, locale)}
              compactValue={fmtMoneyCompact(money(summary.total_fees), currency, locale)}
            />
            <Meta
              metric="profitFactor"
              label={tr({ id: "accounts.pf", message: "PF" })}
              value={summary.profit_factor != null ? summary.profit_factor.toFixed(2) : "—"}
            />
            <Meta
              label={tr({ id: "accounts.expect", message: "Expect" })}
              value={fmtSignedMoney(money(summary.expectancy), currency, locale)}
              compactValue={fmtSignedMoneyCompact(money(summary.expectancy), currency, locale)}
              className={pnlColor(summary.expectancy)}
            />
          </div>
        </div>
      </section>

      {/* Outcome filters */}
      <div className="grid shrink-0 auto-rows-[minmax(91px,auto)] grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-2 xl:grid-cols-4">
        <StatBar
          label={tr({ id: "accounts.wins", message: "Wins" })}
          value={String(summary.wins)}
          sub={fmtPct(summary.win_rate, locale)}
          tone="pos"
          active={tradeStatusFilter === "win"}
          onClick={toggle ? () => toggle("win") : undefined}
        />
        <StatBar
          label={tr({ id: "accounts.losses", message: "Losses" })}
          value={String(summary.losses)}
          sub={fmtPct(summary.losses / total, locale)}
          tone="neg"
          active={tradeStatusFilter === "loss"}
          onClick={toggle ? () => toggle("loss") : undefined}
        />
        <StatBar
          label={tr({ id: "accounts.open", message: "Open" })}
          value={String(openCount)}
          sub={fmtPct(openCount / allTotal, locale)}
          tone="accent"
          active={tradeStatusFilter === "open"}
          onClick={toggle ? () => toggle("open") : undefined}
        />
        <StatBar
          label={tr({ id: "accounts.wash", message: "Wash" })}
          value={String(summary.breakeven)}
          sub={fmtPct(summary.breakeven / total, locale)}
          tone="amber"
          active={tradeStatusFilter === "wash"}
          onClick={toggle ? () => toggle("wash") : undefined}
        />
      </div>

      {/* Avg edge */}
      <div className="grid min-h-[91px] shrink-0 grid-cols-2 gap-3">
        <StatBar
          label={tr({ id: "accounts.avgWin", message: "Avg win" })}
          value={fmtMoney(money(summary.avg_win), currency, locale)}
          compactValue={fmtMoneyCompact(money(summary.avg_win), currency, locale)}
          tone="pos"
        />
        <StatBar
          label={tr({ id: "accounts.avgLoss", message: "Avg loss" })}
          value={fmtMoney(money(summary.avg_loss), currency, locale)}
          compactValue={fmtMoneyCompact(money(summary.avg_loss), currency, locale)}
          tone="neg"
        />
      </div>
    </div>
  );
}
