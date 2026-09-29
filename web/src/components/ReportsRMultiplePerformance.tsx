import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { RSummary } from "@/lib/api/types";
import { usePrivacyMode } from "@/lib/displayPrefs";
import { Card } from "./Card";
import { ChartFrame, chartTheme, chartTooltipStyle } from "./ChartFrame";
import { EmptyState } from "./EmptyState";
import { Skeleton } from "./Skeleton";
import { StatCard } from "./StatCard";

export interface ReportsRMultiplePerformanceProps {
  rSummary?: RSummary;
  loading: boolean;
  error: boolean;
}

const POS = "var(--profit)";
const NEG = "var(--loss)";

function formatR(v: number): string {
  return `${v >= 0 ? "+" : ""}${v.toFixed(2)}R`;
}

export function ReportsRMultiplePerformance({
  rSummary,
  loading,
  error,
}: ReportsRMultiplePerformanceProps) {
  const { t: localize } = useSecondaryLingui();

  usePrivacyMode();
  // `total_trades` here is already the R-eligible (included) count — `excluded`
  // is a disjoint count of trades skipped for missing risk, not a subset of it.
  const included = rSummary?.total_trades ?? 0;
  const distribution = rSummary?.distribution ?? [];
  const excluded = rSummary?.excluded ?? 0;
  const hasData = Boolean(rSummary && included > 0);

  return (
    <Card title={localize({ id: "reports.rMultiples", message: "R-Multiples" })}>
      {loading ? (
        <Skeleton height="280px" />
      ) : error ? (
        <p className="text-xs text-destructive">
          {localize({
            id: "reports.failedToLoadRMultiplePerformance",
            message: "Failed to load R-multiple performance.",
          })}
        </p>
      ) : !hasData ? (
        <EmptyState
          title={localize({ id: "reports.noRData", message: "No R data" })}
          hint={localize({
            id: "reports.setStopsOnYourTradesToSeeRMultiples",
            message: "Set stops on your trades to see R-multiples.",
          })}
        />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <StatCard
              variant="bento"
              align="center"
              label={localize({ id: "reports.avgRTrade", message: "Avg R/Trade" })}
              value={formatR(rSummary!.avg_r)}
              accent={rSummary!.avg_r >= 0 ? "pos" : "neg"}
              hint={localize({
                id: "reports.value0OfValue1Trades",
                message: `${{ value0: included }} of ${{ value1: included + excluded }} trades`,
              })}
            />
            <StatCard
              variant="bento"
              align="center"
              label={localize({ id: "reports.avgWinningR", message: "Avg Winning R" })}
              value={formatR(rSummary!.avg_win_r)}
              accent="pos"
            />
            <StatCard
              variant="bento"
              align="center"
              label={localize({ id: "reports.avgLosingR", message: "Avg Losing R" })}
              value={formatR(rSummary!.avg_loss_r)}
              accent="neg"
            />
            <StatCard
              variant="bento"
              align="center"
              label={localize({ id: "reports.bestWorstR", message: "Best / Worst R" })}
              value={`${formatR(rSummary!.best_r)} / ${formatR(rSummary!.worst_r)}`}
            />
          </div>

          {distribution.length > 0 ? (
            <div className="mt-4">
              <ChartFrame className="rounded-none border-0">
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart
                    data={distribution.map((bucket) => ({
                      ...bucket,
                      label: bucket.label.replace(" to ", " – "),
                    }))}
                    margin={{ top: 12, right: 16, bottom: 0, left: 0 }}
                  >
                    <CartesianGrid vertical={false} stroke={chartTheme.gridColor} />
                    <XAxis
                      dataKey="label"
                      tick={{ fontSize: 10, fill: chartTheme.axisColor }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <YAxis
                      tick={{ fontSize: 10, fill: chartTheme.axisColor }}
                      axisLine={false}
                      tickLine={false}
                      width={32}
                      allowDecimals={false}
                    />
                    <Tooltip
                      {...chartTooltipStyle}
                      formatter={(value) => {
                        const count = Number(value ?? 0);
                        const pct = included > 0 ? ((count / included) * 100).toFixed(0) : "0";
                        return [`${count} (${pct}%)`, "Trades"];
                      }}
                      cursor={{ fill: chartTheme.cursorFill }}
                    />
                    <Bar dataKey="count" radius={[2, 2, 0, 0]}>
                      {distribution.map((b) => (
                        <Cell key={b.label} fill={b.from < 0 ? NEG : POS} fillOpacity={0.85} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </ChartFrame>
              <p className="mt-2 text-[11px] text-muted-foreground">
                {localize({
                  id: "reports.includedClosedTrades",
                  message: `Showing ${{ included: included }} of ${{ total: included + excluded }} closed trades`,
                })}
                {excluded > 0
                  ? localize({
                      id: "reports.value0ExcludedNoStop",
                      message: `, ${{ value0: excluded }} excluded (no stop)`,
                    })
                  : ""}
              </p>
            </div>
          ) : null}
        </>
      )}
    </Card>
  );
}
