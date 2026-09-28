import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { HomeView } from "@/app/screens/HomeView";
import type { HomeBreakdownDim } from "@/components/HomeBreakdownChart";
import { TradeDetailSheet } from "@/components/TradeDetailSheet";
import { ytdFiltersForYear } from "@/lib/annualGoal";
import { buildDayRecords, dayKeyInTz } from "@/lib/calendar";
import { normalizeFilterDate, useFilterParams, useFilters } from "@/lib/filters";
import { computeHeaderStats } from "@/lib/headerStats";
import { useAccounts } from "@/lib/hooks/useAccounts";
import {
  useAccountValue,
  useBreakdown,
  useDailyPnl,
  useEquityCurve,
  useSummary,
} from "@/lib/hooks/useAnalytics";
import { useAnnualGoal, useClearAnnualGoal, useSaveAnnualGoal } from "@/lib/hooks/useAnnualGoal";
import { useCash } from "@/lib/hooks/useCash";
import { useTrades } from "@/lib/hooks/useTrades";
import { filterTradesByStatus } from "@/lib/tradeFilters";
import { useUI } from "@/lib/ui";
import type { AccountValueWarning } from "@/lib/api/types";

const IGNORED_PRICES_KEY = "tradermemos-account-value-ignored-prices";

export const Route = createFileRoute("/home")({
  component: HomePage,
});

function monthRange(year: number, month: number, tz: string) {
  const pad = (n: number) => String(n).padStart(2, "0");
  const lastDay = new Date(Date.UTC(year, month, 0)).getUTCDate();
  return {
    from: normalizeFilterDate(`${year}-${pad(month)}-01`, "start", tz),
    to: normalizeFilterDate(`${year}-${pad(month)}-${pad(lastDay)}`, "end", tz),
  };
}

function HomePage() {
  const filters = useFilterParams();
  const accountIds = useFilters((s) => s.accountIds);
  const tradeStatusFilter = useFilters((s) => s.tradeStatus);
  const toggleTradeStatus = useFilters((s) => s.toggleTradeStatus);
  const setSymbol = useFilters((s) => s.setSymbol);
  const navigate = useNavigate();
  const openModal = useUI((s) => s.openModal);
  const [selectedTradeId, setSelectedTradeId] = useState<string | null>(null);
  const [breakdownDim, setBreakdownDim] = useState<HomeBreakdownDim>("day_of_week");
  const [accountValueRange, setAccountValueRange] = useState("30D");
  const [ignoredPrices, setIgnoredPrices] = useState<string[]>(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(IGNORED_PRICES_KEY) ?? "[]");
      return Array.isArray(saved) && saved.every((item) => typeof item === "string") ? saved : [];
    } catch {
      return [];
    }
  });

  const now = new Date();
  const calendarYear = now.getFullYear();
  const calendarMonth = now.getMonth() + 1;
  const range = monthRange(calendarYear, calendarMonth, filters.tz);
  const monthFilters = { ...filters, from: range.from, to: range.to };
  const ytdFilters = useMemo(
    () => ytdFiltersForYear(filters, calendarYear),
    [filters, calendarYear],
  );

  const summaryQ = useSummary(filters);
  const ytdSummaryQ = useSummary(ytdFilters);
  const equityQ = useEquityCurve(filters);
  const accountsQ = useAccounts();
  const accountValueQ = useAccountValue(
    accountValueFilters(filters.account_id, accountValueRange, ignoredPrices),
    accountsQ.data ?? [],
  );
  const tradesQ = useTrades(filters);
  const monthTradesQ = useTrades(monthFilters);
  const cashQ = useCash(filters);
  const dailyQ = useDailyPnl(monthFilters);
  const breakdownQ = useBreakdown(breakdownDim, filters);
  const annualGoalQ = useAnnualGoal(calendarYear);
  const saveAnnualGoalM = useSaveAnnualGoal();
  const clearAnnualGoalM = useClearAnnualGoal();

  const trades = filterTradesByStatus(
    [...(tradesQ.data ?? [])].sort(
      (a, b) => new Date(b.opened_at).getTime() - new Date(a.opened_at).getTime(),
    ),
    tradeStatusFilter,
  );
  const calendarDayRecords = useMemo(
    () => buildDayRecords(monthTradesQ.data ?? [], "close", filters.tz),
    [monthTradesQ.data, filters.tz],
  );

  const headerStats = computeHeaderStats({
    accounts: accountsQ.data ?? [],
    accountIds,
    cashTx: cashQ.data ?? [],
    summary: summaryQ.data,
    trades: tradesQ.data ?? [],
  });

  return (
    <>
      <HomeView
        summaryLoading={summaryQ.isLoading}
        summaryError={summaryQ.isError}
        summary={summaryQ.data}
        equityLoading={equityQ.isLoading}
        equityError={equityQ.isError}
        equityPoints={equityQ.data?.points ?? []}
        maxDrawdown={equityQ.data?.max_drawdown}
        accountValue={accountValueQ.data}
        accountValueLoading={accountsQ.isLoading || accountValueQ.isLoading}
        accountValueError={accountsQ.isError || accountValueQ.isError}
        accountValueRange={accountValueRange}
        onAccountValueRangeChange={setAccountValueRange}
        onIgnoreAccountValueWarning={(warning: AccountValueWarning) => {
          if (!warning.instrument) return;
          const key = `${warning.instrument}:${warning.date}`;
          const next = ignoredPrices.includes(key) ? ignoredPrices : [...ignoredPrices, key];
          localStorage.setItem(IGNORED_PRICES_KEY, JSON.stringify(next));
          setIgnoredPrices(next);
        }}
        tradesLoading={tradesQ.isLoading}
        tradesError={tradesQ.isError}
        trades={trades}
        baselineTrades={tradesQ.data ?? []}
        accounts={accountsQ.data ?? []}
        selectedAccountIds={accountIds}
        tradeStatusFilter={tradeStatusFilter}
        onToggleTradeStatus={toggleTradeStatus}
        onSelectTrade={(t) => setSelectedTradeId(t.id)}
        onOpenFullPage={(t) => void navigate({ to: "/trades/$id", params: { id: t.id } })}
        onFilterSymbol={(symbol) => setSymbol(symbol)}
        onDeleted={(t) => {
          if (selectedTradeId === t.id) setSelectedTradeId(null);
        }}
        onViewAllTrades={() => void navigate({ to: "/trades" })}
        onOpenCalendar={() => void navigate({ to: "/calendar" })}
        onOpenReports={() =>
          void navigate({
            to: "/reports",
            search: {
              tab: "overview",
              side: "all",
              dur: "all",
              pnl: "net",
              unit: "abs",
              avg: "mean",
            },
          })
        }
        calendarYear={calendarYear}
        calendarMonth={calendarMonth}
        dailyPnl={dailyQ.data ?? {}}
        todayNetPnl={dailyQ.data?.[dayKeyInTz(now.toISOString(), filters.tz)] ?? 0}
        dayRecords={calendarDayRecords}
        dailyLoading={dailyQ.isLoading}
        dailyError={dailyQ.isError}
        breakdownDim={breakdownDim}
        onBreakdownDimChange={setBreakdownDim}
        breakdown={breakdownQ.data ?? []}
        breakdownLoading={breakdownQ.isLoading}
        breakdownError={breakdownQ.isError}
        accountFunded={headerStats.cash > 0}
        onImport={() => navigate({ to: "/import" })}
        onNewTrade={() => openModal("new-trade")}
        goalYear={calendarYear}
        goalAmount={annualGoalQ.data?.amount}
        goalLoading={annualGoalQ.isLoading}
        goalSaving={saveAnnualGoalM.isPending || clearAnnualGoalM.isPending}
        ytdNetPnl={ytdSummaryQ.data?.net_pnl}
        ytdLoading={ytdSummaryQ.isLoading}
        onSaveGoal={async (amount) => {
          await saveAnnualGoalM.mutateAsync({ year: calendarYear, amount });
        }}
        onClearGoal={async () => {
          await clearAnnualGoalM.mutateAsync(calendarYear);
        }}
      />
      <TradeDetailSheet tradeId={selectedTradeId} onClose={() => setSelectedTradeId(null)} />
    </>
  );
}

function accountValueFilters(
  accountId: string | undefined,
  range: string,
  ignoredPrices: string[],
) {
  const ignored_missing_prices = ignoredPrices.join(",") || undefined;
  if (range === "ALL") return { account_id: accountId, ignored_missing_prices };
  const from = new Date();
  from.setDate(from.getDate() - (range === "30D" ? 29 : 89));
  return { account_id: accountId, from: from.toLocaleDateString("en-CA"), ignored_missing_prices };
}
