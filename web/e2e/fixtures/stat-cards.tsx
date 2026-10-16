import { ReportsPnlHeatmap } from "@/components/ReportsPnlHeatmap";
import { ReportsPeriodReturns } from "@/components/ReportsPeriodReturns";
import { ReportsHourlyList } from "@/components/ReportsHourlyList";
import { ReportsExecutionGrade } from "@/components/ReportsExecutionGrade";
// Browser-only regression fixture; the product build does not include this entry.
import { useState } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { createRootRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { I18nProvider, useLocale } from "@/i18n";
import { TooltipProvider } from "@/components/ui/tooltip";
import { HomeInsightBento } from "@/components/HomeInsightBento";
import { PerformanceStrip } from "@/components/PerformanceStrip";
import { ReportsSummaryBento } from "@/components/ReportsSummaryBento";
import { ReportsDayStrip } from "@/components/ReportsDayStrip";
import { ReportsDisplayProvider } from "@/components/ReportsDisplayContext";
import { ReportsMonteCarlo } from "@/components/ReportsMonteCarlo";
import { ReportsRuleCompliance } from "@/components/ReportsRuleCompliance";
import { ReportsRMultiplePerformance } from "@/components/ReportsRMultiplePerformance";
import { ReportsRiskDrawdown } from "@/components/ReportsRiskDrawdown";
import { BehaviorLossAversionCard } from "@/components/BehaviorLossAversionCard";
import { BehaviorRevengeCard } from "@/components/BehaviorRevengeCard";
import { BehaviorOverconfidenceCard } from "@/components/BehaviorOverconfidenceCard";
import { AnnualGoalCard } from "@/components/AnnualGoalCard";
import { DailyLossCard } from "@/components/DailyLossCard";
import { PropStatusCard } from "@/components/PropStatusCard";
import { TradeSummaryCard } from "@/components/TradeSummaryCard";
import { TradePlanCard } from "@/components/TradePlanCard";
import { TradeExecutionsCard } from "@/components/TradeExecutionsCard";
import { RiskRewardPanel } from "@/components/RiskRewardPanel";
import {
  TradeResultPreview,
  AfterSaveResultPreview,
  BatchTradeResultPreview,
} from "@/components/TradeResultPreview";
import { ShareCardSvg } from "@/components/ShareCard";
import { Money } from "@/components/ResponsiveValue";
import { StatCard } from "@/components/StatCard";
import { StatCell } from "@/components/StatCell";
import { useDisplayPrefs } from "@/lib/displayPrefs";
import { useMoneyFormatters } from "@/lib/useMoneyFormatters";
import { analyticsApi } from "@/lib/api/analytics";
import { accountsApi } from "@/lib/api/accounts";
import { tradesApi } from "@/lib/api/trades";
import { computeTradeInsights } from "@/lib/tradeInsights";
import { buildTradeShareCard } from "@/lib/shareCard";
import type { TradePnlPreview } from "@/lib/tradePnlPreview";
import "@/global.css";

function Audit() {
  const { locale, setLocale } = useLocale();
  const [group, setGroup] = useState("home");
  const [amount, setAmount] = useState(12345678);
  const [currency, setCurrency] = useState("JPY");
  const [pnlMode, setPnlMode] = useState<"net" | "gross">("net");
  const [unitMode, setUnitMode] = useState<"abs" | "pct">("abs");
  const [avgMode, setAvgMode] = useState<"mean" | "median">("mean");
  const [goal, setGoal] = useState<number | null>(10000000);
  const [showAmounts, setShowAmounts] = useState(true);
  const privacy = useDisplayPrefs((s) => s.privacyMode);
  const formatters = useMoneyFormatters();
  const q = useQuery({
    queryKey: ["stat-card-audit"],
    queryFn: async () => {
      const accounts = await accountsApi.list();
      const account = accounts.find((a) => a.name === "Synthetic JPY card QA")!;
      const f = { account_id: account.id, target_currency: "JPY" };
      const [summary, trades, equity, simulation, report, behavior, rSummary, hourly, grades] =
        await Promise.all([
          analyticsApi.summary(f),
          tradesApi.list(f),
          analyticsApi.equityCurve(f),
          analyticsApi.monteCarlo(f),
          analyticsApi.compliance(f),
          analyticsApi.behavior(f),
          analyticsApi.rSummary(f),
          analyticsApi.breakdown("hour_of_day", f),
          analyticsApi.breakdown("trade_quality", f),
        ]);
      const trade = await tradesApi.get(
        trades.trades.find((t) => t.symbol.startsWith("SYNTHETIC_LONG"))!.id,
      );
      return {
        accounts,
        summary,
        trades: trades.trades,
        equity,
        simulation,
        report,
        behavior,
        rSummary,
        hourly,
        grades,
        trade,
      };
    },
  });
  if (!q.data) return <p>{q.error ? String(q.error) : "Loading real disposable API…"}</p>;
  const d = q.data;
  const insights = computeTradeInsights(d.trade);
  const preview: TradePnlPreview = {
    avgEntry: 1234567,
    avgExit: 2345678,
    positionQty: 123456789,
    net: amount,
    gross: amount + 12345,
    feesTotal: 12345,
    rMultiple: 12345.67,
    closed: false,
  };
  return (
    <main className="p-3 space-y-4 min-w-0" data-testid="audit">
      <nav className="flex flex-wrap gap-2 text-xs">
        <label>
          Surface{" "}
          <select aria-label="Surface" value={group} onChange={(e) => setGroup(e.target.value)}>
            {["home", "reports", "callers", "goals", "trade", "primitives", "export"].map((g) => (
              <option key={g}>{g}</option>
            ))}
          </select>
        </label>
        <label>
          Locale{" "}
          <select
            aria-label="Locale"
            value={locale}
            onChange={(e) => void setLocale(e.target.value)}
          >
            {["en", "zh-CN", "zh-HK", "ja", "ko"].map((l) => (
              <option key={l}>{l}</option>
            ))}
          </select>
        </label>
        <label>
          Currency{" "}
          <select
            aria-label="Currency"
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
          >
            <option>JPY</option>
            <option>USD</option>
          </select>
        </label>
        <label>
          Amount{" "}
          <input
            aria-label="Amount"
            type="number"
            className="w-28"
            value={amount}
            onChange={(e) => setAmount(Number(e.target.value))}
          />
        </label>
        <button onClick={() => useDisplayPrefs.getState().setPrivacyMode(!privacy)}>
          Privacy {privacy ? "on" : "off"}
        </button>
        <button onClick={() => setShowAmounts((v) => !v)}>
          Show amounts {String(showAmounts)}
        </button>
        <button onClick={() => setPnlMode((v) => (v === "net" ? "gross" : "net"))}>
          P&L {pnlMode}
        </button>
        <button onClick={() => setUnitMode((v) => (v === "abs" ? "pct" : "abs"))}>
          Units {unitMode}
        </button>
        <button onClick={() => setAvgMode((v) => (v === "mean" ? "median" : "mean"))}>
          Average {avgMode}
        </button>
      </nav>
      <p className="wrap-anywhere" data-testid="api-proof">
        Real API: {d.summary.total_trades} trades · {d.trade.symbol}
      </p>
      <ReportsDisplayProvider
        value={{
          pnlMode,
          unitMode,
          avgMode,
          denominator: 123456789,
          currency,
          fxRate: currency === "JPY" ? 1 : 0.01,
        }}
      >
        <div data-testid="surface" className="space-y-4 min-w-0">
          {group === "home" && (
            <>
              <PerformanceStrip summary={d.summary} trades={d.trades} currency={currency} />
              <HomeInsightBento
                summary={d.summary}
                trades={d.trades}
                currency={currency}
                maxDrawdown={9876543}
              />
            </>
          )}
          {group === "reports" && (
            <>
              <ReportsSummaryBento {...d} />
              <div data-testid="heatmap">
                <ReportsPnlHeatmap trades={d.trades} loading={false} error={false} />
              </div>
              <ReportsPeriodReturns
                trades={d.trades}
                loading={false}
                currency={currency}
                fxRate={currency === "JPY" ? 1 : 0.01}
                denominator={123456789}
              />
              <ReportsHourlyList breakdown={d.hourly.groups} loading={false} error={false} />
              <ReportsExecutionGrade breakdown={d.grades.groups} loading={false} error={false} />
              <ReportsDayStrip trades={d.trades} />
            </>
          )}
          {group === "callers" && (
            <>
              <ReportsRiskDrawdown
                trades={d.trades}
                equityPoints={d.equity.points}
                loading={false}
                error={false}
              />
              <ReportsMonteCarlo
                simulation={d.simulation}
                loading={false}
                error={false}
                currency={currency}
              />
              <ReportsRuleCompliance report={d.report} loading={false} error={false} />
              <ReportsRMultiplePerformance rSummary={d.rSummary} loading={false} error={false} />
              <BehaviorLossAversionCard report={d.behavior} loading={false} error={false} />
              <BehaviorRevengeCard report={d.behavior} loading={false} error={false} />
              <BehaviorOverconfidenceCard report={d.behavior} loading={false} error={false} />
            </>
          )}
          {group === "goals" && (
            <>
              <AnnualGoalCard
                year={2026}
                goalAmount={goal}
                ytdNetPnl={amount}
                currency={currency}
                onSave={async (n) => setGoal(n)}
                onClear={async () => setGoal(null)}
              />
              <AnnualGoalCard
                variant="compact"
                year={2026}
                goalAmount={goal}
                ytdNetPnl={amount}
                currency={currency}
                onSave={async (n) => setGoal(n)}
              />
              <DailyLossCard todayNetPnl={amount} currency={currency} />
              <PropStatusCard
                accounts={d.accounts}
                selectedAccountId={d.accounts.find((a) => a.account_type === "prop")?.id}
              />
            </>
          )}
          {group === "trade" && (
            <>
              <TradeSummaryCard trade={d.trade} insights={insights} />
              <TradePlanCard trade={d.trade} insights={insights} />
              <RiskRewardPanel trade={d.trade} />
              <TradeExecutionsCard trade={d.trade} />
              <TradeResultPreview preview={preview} currency={currency} locale={locale} />
              <AfterSaveResultPreview
                preview={preview}
                entryTotal={123456789}
                accountNetPnl={amount}
                accountCash={123456789}
                depositedCapital={123456789}
                currency={currency}
                locale={locale}
              />
              <BatchTradeResultPreview
                batch={{
                  symbolCount: 1000,
                  withFills: 1000,
                  closedCount: 999,
                  openCount: 1,
                  net: amount,
                  feesTotal: 12345,
                  riskTotal: 1234567,
                  rMultiple: 12345.67,
                }}
                currency={currency}
                locale={locale}
              />
            </>
          )}
          {group === "primitives" && (
            <div className="grid grid-cols-2 gap-3">
              <StatCard
                label="Synthetic long count"
                metric="winRate"
                value="123456789012345"
                hint="Long exact count and explanatory qualifier"
              />
              <StatCard
                label="Synthetic long symbol"
                value="SYNTHETIC_UNBROKEN_SYMBOL_ABCDEFGHIJKLMNOPQRSTUVWXYZ"
              />
              <StatCell label="Synthetic money and hint" hint="123456789 contracts × 100">
                <Money value={amount} currency={currency} signed />
              </StatCell>
              <div className="text-[38px]" data-testid="boundary-money">
                <Money value={amount} currency={currency} signed />
              </div>
            </div>
          )}
          {group === "export" && (
            <ShareCardSvg
              data={buildTradeShareCard(
                d.trade,
                {
                  ...insights,
                  setupName: "SYNTHETIC_SETUP_ABCDEFGHIJKLMNOPQRSTUVWXYZ_ABCDEFGHIJKLMN",
                },
                { showAmounts, locale, ...formatters },
              )}
            />
          )}
        </div>
      </ReportsDisplayProvider>
    </main>
  );
}
const route = createRootRoute({ component: Audit });
const router = createRouter({ routeTree: route });
createRoot(document.getElementById("root")!).render(
  <QueryClientProvider client={new QueryClient()}>
    <I18nProvider>
      <TooltipProvider>
        <RouterProvider router={router} />
      </TooltipProvider>
    </I18nProvider>
  </QueryClientProvider>,
);
