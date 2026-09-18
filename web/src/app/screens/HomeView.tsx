import { AlertTriangle, ArrowRight, Plus, TrendingUp, Upload } from "lucide-react";
import { useMemo, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { AnnualGoalCard } from "@/components/AnnualGoalCard";
import { DailyLossCard } from "@/components/DailyLossCard";
import { PropStatusCard } from "@/components/PropStatusCard";
import { Card } from "@/components/Card";
import { ChartFrame, chartTheme, chartTooltipStyle } from "@/components/ChartFrame";
import {
  Collapsible,
  CollapsibleChevron,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/Collapsible";
import { HomeAccountContribution } from "@/components/HomeAccountContribution";
import { type HomeBreakdownDim, HomeBreakdownChart } from "@/components/HomeBreakdownChart";
import { HomeInsightBento } from "@/components/HomeInsightBento";
import { HomeMiniCalendar } from "@/components/HomeMiniCalendar";
import { DataTable } from "@/components/DataTable";
import { ItemGroup } from "@/components/Item";
import { EmptyState } from "@/components/EmptyState";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Page } from "@/components/Page";
import { PerformanceStrip } from "@/components/PerformanceStrip";
import { SegmentedControl } from "@/components/SegmentedControl";
import { Skeleton } from "@/components/Skeleton";
import { CardSkeleton } from "@/components/skeletons/card-skeleton";
import { TableSkeleton } from "@/components/skeletons/table-skeleton";
import { tradeColumns } from "@/components/tradeColumns";
import { TradeListItem } from "@/components/TradeListItem";
import type {
  Account,
  AccountValue,
  AccountValuePoint,
  AccountValueWarning,
  BreakGroup,
  EquityPoint,
  Summary,
  Trade,
} from "@/lib/api/types";
import type { DayRecord } from "@/lib/calendar";
import { uniqueDayTicks } from "@/lib/chartTicks";
import { accountBaseCurrency, useDisplayTimePrefs, usePrivacyMode } from "@/lib/displayPrefs";
import { soleAccountId } from "@/lib/filters";
import { COMPACT_VIEWPORT, useMediaQuery } from "@/lib/hooks/use-mobile";
import { useMoneyFx } from "@/lib/hooks/useMoneyFx";
import { fmtDayShort, fmtMoney, fmtMoneyCompact } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import type { TradeStatusFilter } from "@/lib/tradeFilters";

export interface HomeViewProps {
  summaryLoading: boolean;
  summaryError: boolean;
  summary: Summary | undefined;
  equityLoading: boolean;
  equityError: boolean;
  equityPoints: EquityPoint[];
  maxDrawdown?: number;
  accountValue: AccountValue | undefined;
  accountValueLoading: boolean;
  accountValueError: boolean;
  accountValueRange: string;
  onAccountValueRangeChange: (range: string) => void;
  onIgnoreAccountValueWarning: (warning: AccountValueWarning) => void;
  tradesLoading: boolean;
  tradesError: boolean;
  trades: Trade[];
  accounts: Account[];
  selectedAccountIds: string[] | undefined;
  tradeStatusFilter?: TradeStatusFilter;
  onToggleTradeStatus?: (filter: TradeStatusFilter) => void;
  onSelectTrade: (t: Trade) => void;
  onOpenFullPage: (t: Trade) => void;
  onFilterSymbol?: (symbol: string) => void;
  onDeleted?: (t: Trade) => void;
  onViewAllTrades: () => void;
  onOpenCalendar: () => void;
  onOpenReports: () => void;
  calendarYear: number;
  calendarMonth: number;
  dailyPnl: Record<string, number>;
  /** Today's realized net P&L (account base currency) for the daily-loss tracker. */
  todayNetPnl?: number;
  dayRecords?: Record<string, DayRecord>;
  dailyLoading: boolean;
  dailyError: boolean;
  breakdownDim: HomeBreakdownDim;
  onBreakdownDimChange: (dim: HomeBreakdownDim) => void;
  breakdown: BreakGroup[];
  breakdownLoading: boolean;
  breakdownError: boolean;
  accountFunded: boolean;
  onImport: () => void;
  onNewTrade: () => void;
  goalYear: number;
  goalAmount: number | null | undefined;
  goalLoading: boolean;
  goalSaving: boolean;
  ytdNetPnl: number | undefined;
  ytdLoading: boolean;
  onSaveGoal: (amount: number) => Promise<void>;
  onClearGoal: () => Promise<void>;
}

const RANGES = [
  { value: "30D", label: "30D" },
  { value: "90D", label: "90D" },
  { value: "ALL", label: "ALL" },
];

/** Recent trades shown on the overview — full log lives on Trades. */
export const HOME_RECENT_LIMIT = 10;

function rangeCutoff(range: string): number | null {
  if (range === "ALL") return null;
  const days = range === "30D" ? 30 : 90;
  return Date.now() - days * 86400_000;
}

function EquityCurveChart({
  equityLoading,
  equityError,
  equityPoints,
  currency,
  fxRate = 1,
  range,
}: {
  equityLoading: boolean;
  equityError: boolean;
  equityPoints: EquityPoint[];
  currency: string;
  fxRate?: number;
  range: string;
}) {
  usePrivacyMode();
  useDisplayTimePrefs();
  const cutoff = rangeCutoff(range);
  const visible = useMemo(() => {
    const filtered = cutoff
      ? equityPoints.filter((p) => new Date(p.at).getTime() >= cutoff)
      : equityPoints;
    if (fxRate === 1) return filtered;
    return filtered.map((p) => ({ ...p, equity: p.equity * fxRate }));
  }, [cutoff, equityPoints, fxRate]);
  const dayTicks = useMemo(() => uniqueDayTicks(visible), [visible]);

  if (equityLoading) {
    return <Skeleton className="min-h-[240px] w-full flex-1 sm:min-h-[280px]" />;
  }
  if (equityError) {
    return <p className="text-xs text-destructive">Failed to load equity curve.</p>;
  }
  if (visible.length === 0) {
    return <EmptyState title="No equity data" />;
  }

  return (
    <div className="relative min-h-[240px] w-full flex-1 sm:min-h-[280px] lg:min-h-0">
      <ChartFrame inset className="absolute inset-0 rounded-none border-0 bg-transparent">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={visible} margin={{ top: 8, right: 8, bottom: 0, left: 4 }}>
            <defs>
              <linearGradient id="eq-fill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={chartTheme.accentStroke} stopOpacity={0.25} />
                <stop offset="95%" stopColor={chartTheme.accentStroke} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} stroke={chartTheme.gridColor} />
            <XAxis
              dataKey="at"
              ticks={dayTicks}
              tick={{ fontSize: 10, fill: chartTheme.axisColor }}
              tickFormatter={(v: string) => fmtDayShort(v, intlLocale())}
              axisLine={false}
              tickLine={false}
              minTickGap={60}
            />
            <YAxis
              tick={{ fontSize: 10, fill: chartTheme.axisColor }}
              tickFormatter={(v: number) => fmtMoneyCompact(v, currency, intlLocale())}
              axisLine={false}
              tickLine={false}
              width={52}
              domain={["auto", "auto"]}
            />
            <Tooltip
              {...chartTooltipStyle}
              labelFormatter={(label) => String(label ?? "").slice(0, 10)}
              formatter={(value) => [
                fmtMoney(Number(value ?? 0), currency, intlLocale()),
                "Equity",
              ]}
              cursor={{ fill: chartTheme.cursorFill }}
            />
            <Area
              type="monotone"
              dataKey="equity"
              stroke={chartTheme.accentStroke}
              strokeWidth={1.5}
              fill="url(#eq-fill)"
              dot={false}
              activeDot={{ r: 3, fill: chartTheme.accentStroke }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartFrame>
    </div>
  );
}

function AccountValueChart({
  data,
  loading,
  error,
  currency,
  fxRate,
}: {
  data: AccountValue | undefined;
  loading: boolean;
  error: boolean;
  currency: string;
  fxRate: number;
}) {
  const points = useMemo(
    () =>
      (data?.points ?? []).map((point) => ({
        ...point,
        timestamp: new Date(`${point.date}T00:00:00+09:00`).getTime(),
        estimated_account_value:
          point.estimated_account_value == null ? null : point.estimated_account_value * fxRate,
        contributed_capital: point.contributed_capital * fxRate,
      })),
    [data?.points, fxRate],
  );

  if (loading) return <Skeleton className="min-h-[240px] w-full" />;
  if (error) return <p className="text-xs text-destructive">Failed to load account value.</p>;
  if (points.length === 0) {
    return <EmptyState title="No account value data" hint="Import executions or cash activity." />;
  }

  return (
    <div className="h-[280px] w-full">
      <ChartFrame inset className="h-full rounded-none border-0 bg-transparent">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={points} margin={{ top: 8, right: 8, bottom: 0, left: 4 }}>
            <CartesianGrid vertical={false} stroke={chartTheme.gridColor} />
            <XAxis
              dataKey="timestamp"
              type="number"
              scale="time"
              domain={["dataMin", "dataMax"]}
              tick={{ fontSize: 10, fill: chartTheme.axisColor }}
              tickFormatter={(value: number) =>
                fmtDayShort(new Date(value).toISOString(), intlLocale())
              }
              axisLine={false}
              tickLine={false}
              minTickGap={60}
            />
            <YAxis
              tick={{ fontSize: 10, fill: chartTheme.axisColor }}
              tickFormatter={(value: number) => fmtMoneyCompact(value, currency, intlLocale())}
              axisLine={false}
              tickLine={false}
              width={52}
              domain={["auto", "auto"]}
            />
            <Tooltip
              {...chartTooltipStyle}
              labelFormatter={(value) => new Date(Number(value)).toLocaleDateString(intlLocale())}
              formatter={(value, name) => [
                fmtMoney(Number(value ?? 0), currency, intlLocale()),
                name === "estimated_account_value"
                  ? "Estimated Account Value"
                  : "Contributed Capital",
              ]}
              cursor={{ stroke: chartTheme.gridColor }}
            />
            <Line
              type="monotone"
              dataKey="estimated_account_value"
              name="Estimated Account Value"
              stroke={chartTheme.accentStroke}
              strokeWidth={2}
              dot={false}
              connectNulls={false}
            />
            <Line
              type="stepAfter"
              dataKey="contributed_capital"
              name="Contributed Capital"
              stroke="var(--color-chart-2)"
              strokeWidth={1.5}
              strokeDasharray="5 4"
              dot={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </ChartFrame>
    </div>
  );
}

function AccountValueWarnings({
  points,
  onIgnore,
}: {
  points: AccountValuePoint[];
  onIgnore: (warning: AccountValueWarning) => void;
}) {
  const [open, setOpen] = useState(false);
  const warningMap = new Map<string, AccountValueWarning>();
  for (const warning of points.flatMap((point) => point.warnings ?? [])) {
    const key = `${warning.code}\u0000${warning.instrument ?? ""}\u0000${warning.execution_id ?? ""}\u0000${warning.message}`;
    if (!warningMap.has(key)) warningMap.set(key, warning);
  }
  const warnings = Array.from(warningMap.values());
  const incomplete = points.some((point) => point.status !== "complete");
  if (!incomplete && warnings.length === 0) return null;

  if (warnings.length === 0) {
    return (
      <Alert variant="warning">
        <AlertTriangle />
        <AlertTitle>Incomplete reconstruction</AlertTitle>
        <AlertDescription>Some account values are unavailable for this range.</AlertDescription>
      </Alert>
    );
  }

  if (warnings.length === 1) {
    return <AccountValueWarningAlert warning={warnings[0]} onIgnore={onIgnore} />;
  }

  return (
    <Alert variant="warning">
      <AlertTriangle />
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger className="w-full">
          <span className="font-medium">{warnings.length} account value issues</span>
          <span className="text-xs text-muted-foreground">{open ? "Hide" : "Show all"}</span>
          <CollapsibleChevron />
        </CollapsibleTrigger>
        <CollapsibleContent className="pt-2">
          <ul className="grid gap-2">
            {warnings.map((warning) => (
              <li
                key={`${warning.code}-${warning.instrument}-${warning.execution_id}-${warning.message}`}
              >
                <AccountValueWarningDetail warning={warning} onIgnore={onIgnore} />
              </li>
            ))}
          </ul>
        </CollapsibleContent>
      </Collapsible>
    </Alert>
  );
}

function AccountValueWarningAlert({
  warning,
  onIgnore,
}: {
  warning: AccountValueWarning;
  onIgnore: (warning: AccountValueWarning) => void;
}) {
  return (
    <Alert variant="warning">
      <AlertTriangle />
      <AlertTitle>{accountValueWarningTitle(warning.code)}</AlertTitle>
      <AlertDescription>
        <AccountValueWarningDetail warning={warning} onIgnore={onIgnore} />
      </AlertDescription>
    </Alert>
  );
}

function AccountValueWarningDetail({
  warning,
  onIgnore,
}: {
  warning: AccountValueWarning;
  onIgnore: (warning: AccountValueWarning) => void;
}) {
  return (
    <p className="flex items-center gap-2">
      <span className="font-medium text-foreground">
        {[
          warning.instrument,
          warning.date,
          warning.execution_id && `Execution ${warning.execution_id}`,
        ]
          .filter(Boolean)
          .join(" · ")}
      </span>
      {warning.message ? ` — ${warning.message}` : null}
      {warning.code === "missing_price" && warning.instrument ? (
        <Button type="button" variant="outline" size="sm" onClick={() => onIgnore(warning)}>
          Ignore
        </Button>
      ) : null}
    </p>
  );
}

function accountValueWarningTitle(code: string) {
  if (code === "unsupported_corporate_action") return "Corporate action review required";
  if (code === "missing_price") return "Missing market data";
  if (code === "carried_forward_suspension_price") return "Estimated using previous close";
  return "Incomplete reconstruction";
}

export function HomeView({
  summaryLoading,
  summaryError,
  summary,
  equityLoading,
  equityError,
  equityPoints,
  maxDrawdown,
  accountValue,
  accountValueLoading,
  accountValueError,
  accountValueRange,
  onAccountValueRangeChange,
  onIgnoreAccountValueWarning,
  tradesLoading,
  tradesError,
  trades,
  accounts,
  selectedAccountIds,
  tradeStatusFilter,
  onToggleTradeStatus,
  onSelectTrade,
  onOpenFullPage,
  onFilterSymbol,
  onDeleted,
  onViewAllTrades,
  onOpenCalendar,
  onOpenReports,
  calendarYear,
  calendarMonth,
  dailyPnl,
  todayNetPnl = 0,
  dayRecords,
  dailyLoading,
  dailyError,
  breakdownDim,
  onBreakdownDimChange,
  breakdown,
  breakdownLoading,
  breakdownError,
  accountFunded,
  onImport,
  onNewTrade,
  goalYear,
  goalAmount,
  goalLoading,
  goalSaving,
  ytdNetPnl,
  ytdLoading,
  onSaveGoal,
  onClearGoal,
}: HomeViewProps) {
  const baseCurrency = accountBaseCurrency(accounts, selectedAccountIds);
  const { currency, rate } = useMoneyFx(baseCurrency);
  const fxRate = rate ?? 1;
  const compact = useMediaQuery(COMPACT_VIEWPORT);
  const [range, setRange] = useState("30D");
  const [warningToIgnore, setWarningToIgnore] = useState<AccountValueWarning | null>(null);

  const recentTrades = useMemo(() => trades.slice(0, HOME_RECENT_LIMIT), [trades]);
  const hasMoreTrades = trades.length > HOME_RECENT_LIMIT;

  const noData =
    !summaryLoading &&
    !tradesLoading &&
    !summaryError &&
    !tradesError &&
    summary?.total_trades === 0 &&
    trades.length === 0;

  if (noData) {
    const emptyActions = (
      <>
        <Button type="button" variant="default" onClick={onImport}>
          <Upload size={13} strokeWidth={1.75} />
          Import CSV
        </Button>
        <Button type="button" variant="outline" onClick={onNewTrade}>
          <Plus size={13} strokeWidth={1.75} />
          Log trade
        </Button>
      </>
    );

    return (
      <Page fill className="items-center justify-center">
        <EmptyState
          title="No trades yet"
          hint={
            accountFunded
              ? "Account funded — import history or log your first trade to see P&L light up here."
              : "Import broker history or log your first trade to start tracking performance."
          }
          icon={<TrendingUp size={40} strokeWidth={1.5} />}
          actions={emptyActions}
        />
      </Page>
    );
  }

  const recentAction = (
    <Button
      type="button"
      variant="link"
      onClick={onViewAllTrades}
      className="h-auto gap-1 rounded-md text-[11px] font-medium"
    >
      View all trades
      <ArrowRight size={12} strokeWidth={2} aria-hidden />
    </Button>
  );

  return (
    <Page className="min-h-[calc(100svh-52px)]">
      {/* Asymmetric hero: tall equity + performance strip (Signal Terminal bento) */}
      {/* The 320px floor lives on the row track, not as min-h on the container: a
          min-height here makes the container definite and the single auto row stretches
          to exactly fill it, capping the performance strip's taller content so it
          spills out and overlaps the goal card below. minmax() floors without capping. */}
      <div className="grid gap-4 lg:grid-cols-5 lg:grid-rows-[minmax(320px,auto)] lg:items-stretch">
        <Card
          title="Equity curve"
          className="h-full lg:col-span-3"
          fill
          action={
            <SegmentedControl
              ariaLabel="Equity range"
              options={RANGES}
              value={range}
              onChange={setRange}
            />
          }
        >
          {summaryLoading ? (
            <Skeleton className="min-h-[160px] w-full flex-1" />
          ) : summaryError ? (
            <p className="text-xs text-destructive">Failed to load summary.</p>
          ) : !summary ? null : (
            <EquityCurveChart
              equityLoading={equityLoading}
              equityError={equityError}
              equityPoints={equityPoints}
              currency={currency}
              fxRate={fxRate}
              range={range}
            />
          )}
        </Card>

        {summary && !summaryLoading && !summaryError ? (
          <div className="min-h-[265px] lg:col-span-2">
            <PerformanceStrip
              summary={summary}
              trades={trades}
              currency={currency}
              fxRate={fxRate}
              tradeStatusFilter={tradeStatusFilter}
              onToggleTradeStatus={onToggleTradeStatus}
            />
          </div>
        ) : summaryLoading ? (
          <div className="lg:col-span-2">
            <CardSkeleton mediaClassName="h-[210px]" />
          </div>
        ) : null}
      </div>

      <Card
        title="Historical account value"
        action={
          <SegmentedControl
            ariaLabel="Account value range"
            options={RANGES}
            value={accountValueRange}
            onChange={onAccountValueRangeChange}
          />
        }
      >
        <div className="grid gap-3">
          <AccountValueWarnings points={accountValue?.points ?? []} onIgnore={setWarningToIgnore} />
          <AccountValueChart
            data={accountValue}
            loading={accountValueLoading}
            error={accountValueError}
            currency={currency}
            fxRate={fxRate}
          />
          {!accountValueLoading && !accountValueError && (accountValue?.points.length ?? 0) > 0 ? (
            <div className="flex flex-wrap gap-x-5 gap-y-1 text-[11px] text-muted-foreground">
              <span>
                <span className="mr-1.5 inline-block h-0.5 w-4 bg-primary" />
                Estimated Account Value
              </span>
              <span>
                <span className="mr-1.5 inline-block w-4 border-t border-dashed border-chart-2" />
                Contributed Capital
              </span>
            </div>
          ) : null}
        </div>
      </Card>

      <DailyLossCard todayNetPnl={todayNetPnl} currency={currency} fxRate={fxRate} />

      {/* Prop status tracks one funded account; hidden under a multi-account scope. */}
      <PropStatusCard accounts={accounts} selectedAccountId={soleAccountId(selectedAccountIds)} />

      <AnnualGoalCard
        year={goalYear}
        goalAmount={goalAmount}
        ytdNetPnl={ytdNetPnl}
        currency={currency}
        fxRate={fxRate}
        variant="hero"
        loading={goalLoading || ytdLoading}
        saving={goalSaving}
        onSave={onSaveGoal}
        onClear={onClearGoal}
      />

      {summary && !summaryLoading && !summaryError ? (
        <HomeInsightBento
          summary={summary}
          trades={trades}
          currency={currency}
          fxRate={fxRate}
          maxDrawdown={maxDrawdown}
        />
      ) : null}

      <HomeAccountContribution
        trades={trades}
        accounts={accounts}
        currency={currency}
        fxRate={fxRate}
      />

      <div className="grid gap-4 lg:grid-cols-5">
        <div className="lg:col-span-3">
          <HomeBreakdownChart
            dim={breakdownDim}
            onDimChange={onBreakdownDimChange}
            breakdown={breakdown}
            loading={breakdownLoading}
            error={breakdownError}
            currency={currency}
            fxRate={fxRate}
            onOpenReports={onOpenReports}
          />
        </div>
        <div className="lg:col-span-2">
          <HomeMiniCalendar
            year={calendarYear}
            month={calendarMonth}
            dailyPnl={dailyPnl}
            dayRecords={dayRecords}
            currency={currency}
            fxRate={fxRate}
            loading={dailyLoading}
            error={dailyError}
            onOpenCalendar={onOpenCalendar}
          />
        </div>
      </div>

      <Card title="Recent trades" action={recentAction} flush>
        {tradesLoading ? (
          <TableSkeleton rows={5} columns={4} className="m-1" />
        ) : tradesError ? (
          <p className="p-4 text-xs text-destructive">Failed to load trades.</p>
        ) : trades.length === 0 ? (
          <EmptyState
            title={tradeStatusFilter ? "No trades match this filter" : "No trades in this range"}
            hint={tradeStatusFilter ? "Click the stat chip again to clear the filter." : undefined}
          />
        ) : (
          <>
            {/* Phone: the same list row the trade log uses — the wide table
                can't fit, and 10 rows don't need virtualizing. */}
            {compact ? (
              <ItemGroup className="gap-2 px-4 pb-4">
                {recentTrades.map((trade) => (
                  <TradeListItem
                    key={trade.id}
                    trade={trade}
                    currency={currency}
                    fxRate={fxRate}
                    showDate
                    onSelect={onSelectTrade}
                  />
                ))}
              </ItemGroup>
            ) : (
              <DataTable
                columns={tradeColumns(
                  currency,
                  {
                    onOpenDrawer: onSelectTrade,
                    onOpenFullPage,
                    onFilterSymbol,
                    onDeleted,
                  },
                  fxRate,
                )}
                data={recentTrades}
                onRowClick={onSelectTrade}
                maxHeight={360}
              />
            )}
            <p className="shrink-0 py-2 text-center text-xs text-muted-foreground">
              {hasMoreTrades
                ? `Showing ${recentTrades.length} of ${trades.length} trades`
                : `${trades.length} ${trades.length === 1 ? "trade" : "trades"}`}
            </p>
          </>
        )}
      </Card>

      <Dialog
        open={warningToIgnore !== null}
        onOpenChange={(open) => {
          if (!open) setWarningToIgnore(null);
        }}
      >
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Ignore missing market data?</DialogTitle>
            <DialogDescription>
              {warningToIgnore?.instrument} has no unadjusted closing price for{" "}
              {warningToIgnore?.date}. TraderMemos will use the previous available market close for
              this date.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setWarningToIgnore(null)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => {
                if (warningToIgnore) onIgnoreAccountValueWarning(warningToIgnore);
                setWarningToIgnore(null);
              }}
            >
              Ignore and use previous close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Page>
  );
}
