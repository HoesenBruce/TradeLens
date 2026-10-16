import { createContext, useContext, type ReactNode } from "react";
import type { Summary, Trade } from "@/lib/api/types";
import { fmtPct } from "@/lib/format";
import { getIntlLocale } from "@/lib/locale";
import { useLingui } from "@lingui/react";
import { useMoneyFormatters } from "@/lib/useMoneyFormatters";
import { ResponsiveValue } from "./ResponsiveValue";

export type PnlMode = "net" | "gross";
export type UnitMode = "abs" | "pct";
export type AvgMode = "mean" | "median";

export interface ReportsDisplay {
  pnlMode: PnlMode;
  unitMode: UnitMode;
  /** Mean vs outlier-resistant median for per-trade stats; defaults to mean. */
  avgMode?: AvgMode;
  denominator: number; // Net deposits in the same API currency as samples; 0 disables %.
  currency: string;
  fxRate: number;
}

const DEFAULT: ReportsDisplay = {
  pnlMode: "net",
  unitMode: "abs",
  avgMode: "mean",
  denominator: 0,
  currency: "USD",
  fxRate: 1,
};

const Ctx = createContext<ReportsDisplay>(DEFAULT);

export function ReportsMoney({
  value,
  disclosure = true,
}: {
  value: number;
  disclosure?: boolean;
}) {
  const money = useReportsMoney();
  return (
    <ResponsiveValue
      value={money.format(value)}
      compactValue={money.formatCompact(value)}
      disclosure={disclosure}
    />
  );
}

export function ReportsDisplayProvider({
  value,
  children,
}: {
  value: ReportsDisplay;
  children: ReactNode;
}) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useReportsMoney() {
  const d = useContext(Ctx);
  const { i18n } = useLingui();
  const locale = getIntlLocale(i18n.locale);
  // Privacy-bound formatters: their identity changes when privacy flips, so the
  // returned `money` object does too and every consumer's memo cache misses.
  const { privacy, fmtSignedMoney, fmtSignedMoneyCompact } = useMoneyFormatters();
  const pctEnabled = d.denominator > 0;
  const usePct = d.unitMode === "pct" && pctEnabled;

  // Gross = before fees. Do NOT use gross_profit - gross_loss here: those are
  // net-classified win/loss sums, so their difference is identically net_pnl
  // and the toggle would be a no-op. Fall back to net + fees for older APIs.
  const pnl = (s: Summary) =>
    d.pnlMode === "gross" ? (s.gross_pnl ?? s.net_pnl + s.total_fees) : s.net_pnl;
  const tradePnl = (t: Trade) =>
    d.pnlMode === "gross" ? (t.gross_pnl ?? t.net_pnl ?? 0) : (t.net_pnl ?? 0);

  const display = (rawPnl: number) => (usePct ? rawPnl / d.denominator : rawPnl * d.fxRate);
  const format = (rawPnl: number) =>
    usePct
      ? fmtPct(rawPnl / d.denominator, locale)
      : fmtSignedMoney(rawPnl * d.fxRate, d.currency, locale);
  const formatCompact = (rawPnl: number) =>
    usePct
      ? fmtPct(rawPnl / d.denominator, locale)
      : fmtSignedMoneyCompact(rawPnl * d.fxRate, d.currency, locale);
  const formatAxis = (displayValue: number) =>
    usePct ? fmtPct(displayValue, locale) : fmtSignedMoneyCompact(displayValue, d.currency, locale);

  return {
    pnlMode: d.pnlMode,
    avgMode: d.avgMode ?? "mean",
    unitMode: usePct ? "pct" : "abs",
    privacy,
    pctEnabled,
    pnl,
    tradePnl,
    display,
    format,
    formatCompact,
    formatAxis,
  };
}
