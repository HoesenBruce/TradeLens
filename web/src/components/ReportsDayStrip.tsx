import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import type { Trade } from "@/lib/api/types";
import { cn } from "@/lib/cn";
import { useDisplayTimePrefs } from "@/lib/displayPrefs";
import { fmtDayShort, fmtTradeDay } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { useReportsMoney } from "./ReportsDisplayContext";
import { pnlColor } from "./theme-tokens";

export interface DayStripItem {
  date: string;
  pnl: number;
  trades: number;
}

/** Closed-trade daily P&L strip, newest → oldest. */
export function buildReportsDayStrip(
  trades: Trade[],
  tradePnl: (t: Trade) => number = (t) => t.net_pnl ?? 0,
): DayStripItem[] {
  const map = new Map<string, { pnl: number; trades: number }>();
  for (const t of trades) {
    if (t.status === "open") continue;
    const pnl = tradePnl(t);
    if (pnl == null || Number.isNaN(pnl)) continue;
    const date = fmtTradeDay(t.closed_at ?? t.opened_at);
    const cur = map.get(date) ?? { pnl: 0, trades: 0 };
    cur.pnl += pnl;
    cur.trades += 1;
    map.set(date, cur);
  }
  return [...map.entries()]
    .map(([date, s]) => ({
      date,
      pnl: Math.round(s.pnl * 100) / 100,
      trades: s.trades,
    }))
    .sort((a, b) => b.date.localeCompare(a.date));
}

export interface ReportsDayStripProps {
  trades: Trade[];
  loading?: boolean;
  /** Kept for call-site compatibility; display currency/fx come from ReportsDisplayContext. */
  currency?: string;
  fxRate?: number;
  onDayClick?: (date: string) => void;
}

export function ReportsDayStrip({ trades, loading, onDayClick }: ReportsDayStripProps) {
  const { t: localize } = useSecondaryLingui();

  useDisplayTimePrefs();
  const money = useReportsMoney();
  const locale = intlLocale();
  const days = buildReportsDayStrip(trades, money.tradePnl);

  if (loading) {
    return (
      <div className="flex gap-1.5 overflow-hidden">
        {Array.from({ length: 7 }).map((_, i) => (
          <div key={i} className="h-[72px] w-[88px] shrink-0 animate-pulse rounded-lg bg-card" />
        ))}
      </div>
    );
  }

  if (days.length === 0) return null;

  return (
    <section className="min-w-0">
      <p className="mb-2 text-[10px] font-semibold tracking-wide text-chart-3">
        {localize({ id: "accounts.tradingDays", message: "Trading days" })}
      </p>
      <div className="-mx-1 flex gap-1.5 overflow-x-auto px-1 pb-1 [scrollbar-width:thin]">
        {days.map((d) => {
          const dow = new Date(`${d.date}T12:00:00Z`).getUTCDay();
          const isWeekend = dow === 0 || dow === 6;
          const label = fmtDayShort(`${d.date}T12:00:00Z`, locale);
          const CardTag = onDayClick ? "button" : "div";
          const pnlText = money.format(d.pnl);
          const compact = pnlText.length >= 13;
          return (
            <CardTag
              key={d.date}
              type={onDayClick ? "button" : undefined}
              onClick={onDayClick ? () => onDayClick(d.date) : undefined}
              className={cn(
                "flex w-[88px] shrink-0 flex-col rounded-lg bg-card px-2.5 py-2 text-left",
                onDayClick &&
                  "cursor-pointer border-none transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                isWeekend && "opacity-60",
              )}
              aria-label={
                onDayClick
                  ? localize({
                      id: "reports.viewTradesForValue0",
                      message: `View trades for ${{ value0: label }}`,
                    })
                  : undefined
              }
            >
              <p className="text-[10px] font-medium text-muted-foreground">{label}</p>
              <p
                title={pnlText}
                className={cn(
                  "mt-1.5 min-w-0 max-w-full whitespace-nowrap font-semibold leading-none tabular-nums",
                  pnlText.length >= 11
                    ? "text-[9px] tracking-[-0.06em]"
                    : "text-[13px] tracking-[-0.02em]",
                  pnlColor(d.pnl),
                )}
              >
                {compact ? money.formatCompact(d.pnl) : pnlText}
              </p>
              <p className="mt-1 text-[9px] text-muted-foreground">
                {localize({
                  id: "reports.dailyTradeCount",
                  message: `Trades: ${{ count: d.trades }}`,
                })}
              </p>
            </CardTag>
          );
        })}
      </div>
    </section>
  );
}
