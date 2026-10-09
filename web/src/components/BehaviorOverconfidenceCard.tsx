import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import type { BehaviorReport } from "@/lib/api/types";

import { fmtDayShort, fmtPct } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { Card } from "./Card";
import { EmptyState } from "./EmptyState";
import { Pill } from "./Pill";
import { useReportsMoney } from "./ReportsDisplayContext";
import { Skeleton } from "./Skeleton";
import { StatCard } from "./StatCard";

export interface BehaviorOverconfidenceCardProps {
  report?: BehaviorReport;
  loading: boolean;
  error: boolean;
  onSelectTradeId?: (id: string) => void;
}

/**
 * "Do winning streaks make me oversize?" — after 3+ straight wins, flags the
 * next trade when its size jumps 1.5× above the streak's median, then shows
 * how those inflated trades actually did.
 */
export function BehaviorOverconfidenceCard({
  report,
  loading,
  error,
  onSelectTradeId,
}: BehaviorOverconfidenceCardProps) {
  const { t: localize } = useSecondaryLingui();

  const money = useReportsMoney();
  const locale = intlLocale();

  if (loading) {
    return (
      <Card title={localize({ id: "reports.overconfidence", message: "Overconfidence" })}>
        <Skeleton height="160px" />
      </Card>
    );
  }
  if (error) {
    return (
      <Card title={localize({ id: "reports.overconfidence", message: "Overconfidence" })}>
        <p className="m-0 text-xs text-destructive">
          {localize({
            id: "reports.failedToLoadBehaviorReport",
            message: "Failed to load behavior report.",
          })}
        </p>
      </Card>
    );
  }

  const sec = report?.overconfidence;
  if (!sec || sec.events.length === 0) {
    return (
      <Card title={localize({ id: "reports.overconfidence", message: "Overconfidence" })}>
        <EmptyState
          title={localize({
            id: "reports.noPostStreakSizeInflationDetected",
            message: "No post-streak size inflation detected",
          })}
          hint={
            sec && sec.streaks > 0
              ? localize({
                  id: "reports.disciplinedStreaks",
                  message: `Win streaks of 3+ found: ${{ count: sec.streaks }} — position sizing stayed disciplined afterward.`,
                })
              : localize({
                  id: "reports.afterAStreakOf3WinsTheNextTradeIsFlaggedWhen",
                  message:
                    "After a streak of 3+ wins, the next trade is flagged when it is sized 1.5× above the streak's median.",
                })
          }
        />
      </Card>
    );
  }

  const recent = sec.events.slice(-6).reverse();

  return (
    <Card
      title={localize({ id: "reports.overconfidence", message: "Overconfidence" })}
      description={localize({
        id: "reports.sizeInflationAfterWinStreaksTheFlaggedTradesAreTheOnesOpened",
        message:
          "Size inflation after win streaks — the flagged trades are the ones opened bigger right after a hot run.",
      })}
    >
      <div className="flex flex-col gap-5">
        <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
          <StatCard
            label={localize({ id: "reports.winStreaks3", message: "Win streaks (3+)" })}
            value={String(sec.streaks)}
          />
          <StatCard
            label={localize({ id: "reports.inflatedTrades", message: "Inflated trades" })}
            value={String(sec.flagged.trades)}
            hint={localize({
              id: "reports.winRateValue0",
              message: `win rate ${{ value0: fmtPct(sec.flagged.win_rate, locale) }}`,
            })}
          />
          <StatCard
            label={localize({ id: "reports.inflatedPL", message: "Inflated P&L" })}
            value={money.format(sec.flagged.net_pnl)}
            accent={sec.flagged.net_pnl >= 0 ? "pos" : "neg"}
          />
          <StatCard
            label={localize({ id: "reports.baselineWinRate", message: "Baseline win rate" })}
            value={fmtPct(sec.baseline.win_rate, locale)}
            hint={localize({
              id: "reports.value0Trades",
              message: `${{ value0: sec.baseline.trades }} trades`,
            })}
          />
        </div>

        {sec.insufficient_data && (
          <p className="m-0 text-[11px] leading-relaxed text-muted-foreground">
            {localize({
              id: "reports.smallSamplePatternsFirmUpAsMoreClosedTradesAccumulate",
              message: "Small sample — patterns firm up as more closed trades accumulate.",
            })}
          </p>
        )}

        <div className="flex flex-col gap-1.5">
          <p className="m-0 text-[10px] font-semibold tracking-widest text-muted-foreground uppercase">
            {localize({ id: "reports.recentEvents", message: "Recent events" })}
          </p>
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            {recent.map((ev) => (
              <li key={ev.trade_id}>
                <button
                  type="button"
                  onClick={() => onSelectTradeId?.(ev.trade_id)}
                  className="flex w-full items-center justify-between gap-3 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-accent"
                >
                  <span className="flex items-center gap-2">
                    <span className="tabular-nums text-foreground">
                      {fmtDayShort(`${ev.date}T12:00:00Z`, locale)}
                    </span>
                    <span className="font-medium text-foreground">{ev.symbol}</span>
                    <Pill tone="amber">
                      {localize({ id: "reports.size", message: "size ×" })}
                      {(ev.size_ratio ?? 0).toFixed(1)}
                    </Pill>
                  </span>
                  <span
                    className={
                      ev.net_pnl >= 0
                        ? "tabular-nums font-medium text-success"
                        : "tabular-nums font-medium text-destructive"
                    }
                  >
                    {money.format(ev.net_pnl)}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </Card>
  );
}
