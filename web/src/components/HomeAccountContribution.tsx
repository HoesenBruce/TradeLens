import { useLingui as useLinguiMacro } from "@lingui/react/macro";
import type { Account, Trade } from "@/lib/api/types";
import { computeAccountContribution } from "@/lib/homeInsights";
import { cn } from "@/lib/cn";
import { fmtPct, fmtSignedMoney } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { pnlColor } from "./theme-tokens";
import { WinLossRecord } from "./WinLossRecord";
import { usePrivacyMode } from "@/lib/displayPrefs";

export interface HomeAccountContributionProps {
  trades: Trade[];
  accounts: Account[];
  currency: string;
  fxRate?: number;
}

export function HomeAccountContribution({
  trades,
  accounts,
  currency,
  fxRate = 1,
}: HomeAccountContributionProps) {
  const { t: tr } = useLinguiMacro();

  usePrivacyMode();
  const rows = computeAccountContribution(trades, accounts);
  if (rows.length < 2) return null;

  const locale = intlLocale();

  return (
    <section className="rounded-lg bg-card">
      <header className="px-4 py-3">
        <h2 className="text-[10px] font-semibold tracking-wide text-chart-3">
          {tr({ id: "accounts.contribution", message: "Account contribution" })}
        </h2>
      </header>
      <div className="overflow-x-auto px-2 pb-3">
        <table className="w-full min-w-[420px] border-collapse text-left text-[12px]">
          <thead>
            <tr className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              <th className="px-2 py-1.5 font-medium">
                {tr({ id: "accounts.account", message: "Account" })}
              </th>
              <th className="px-2 py-1.5 font-medium tabular-nums">
                {tr({ id: "accounts.trades", message: "Trades" })}
              </th>
              <th className="px-2 py-1.5 font-medium tabular-nums">
                {tr({ id: "accounts.winRate", message: "Win rate" })}
              </th>
              <th className="px-2 py-1.5 font-medium tabular-nums">
                {tr({ id: "accounts.record", message: "Record" })}
              </th>
              <th className="px-2 py-1.5 text-right font-medium tabular-nums">
                {tr({ id: "accounts.pnl", message: "P&L" })}
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.accountId} className="transition-colors hover:bg-accent">
                <td className="px-2 py-2 font-medium text-foreground">{row.name}</td>
                <td className="px-2 py-2 tabular-nums text-muted-foreground">{row.trades}</td>
                <td className="px-2 py-2 tabular-nums text-muted-foreground">
                  {fmtPct(row.winRate, locale)}
                </td>
                <td className="px-2 py-2 tabular-nums text-muted-foreground">
                  <WinLossRecord wins={row.wins} losses={row.losses} separator=" / " />
                </td>
                <td
                  className={cn(
                    "px-2 py-2 text-right font-medium tabular-nums",
                    pnlColor(row.netPnl),
                  )}
                >
                  {fmtSignedMoney(row.netPnl * fxRate, currency, locale)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
