import { t as localize } from "@lingui/core/macro";
export type ReportsTab = "overview" | "win-loss" | "detailed" | "risk" | "behavior";

export interface ReportCardDef {
  id: string;
  label: string;
}

/**
 * Every card the Reports page can render, per tab, in default order. Paired
 * half-width cards (Symbol & Tag, Day of Week & Hourly) count as one unit so
 * toggling can't strand a lone half-width card in a two-column grid.
 */
export const REPORT_CARDS: Record<ReportsTab, ReportCardDef[]> = {
  get overview() {
    return [
      {
        id: "summary",
        get label() {
          return localize({ id: "reports.summaryMetrics", message: "Summary metrics" });
        },
      },
      {
        id: "period-returns",
        get label() {
          return localize({ id: "reports.periodReturns", message: "Period returns" });
        },
      },
      {
        id: "execution-score",
        get label() {
          return localize({
            id: "reports.executionQualityScore",
            message: "Execution quality score",
          });
        },
      },
      {
        id: "playbook",
        get label() {
          return localize({ id: "reports.playbookLeaks", message: "Playbook & Leaks" });
        },
      },
      {
        id: "r-multiple",
        get label() {
          return localize({
            id: "reports.rMultiplePerformance",
            message: "R-Multiple performance",
          });
        },
      },
      {
        id: "execution-grade",
        get label() {
          return localize({ id: "reports.executionGrade2", message: "Execution grade" });
        },
      },
    ];
  },
  get "win-loss"() {
    return [
      {
        id: "rolling-win-rate",
        get label() {
          return localize({ id: "reports.rollingWinRate2", message: "Rolling win rate" });
        },
      },
      {
        id: "metric-evolution",
        get label() {
          return localize({ id: "reports.metricEvolution2", message: "Metric evolution" });
        },
      },
    ];
  },
  get detailed() {
    return [
      {
        id: "session-clock",
        get label() {
          return localize({ id: "reports.sessionClock", message: "Session clock" });
        },
      },
      {
        id: "symbol-tag",
        get label() {
          return localize({ id: "reports.symbolTagBreakdown", message: "Symbol & Tag breakdown" });
        },
      },
      {
        id: "day-hour",
        get label() {
          return localize({ id: "reports.dayOfWeekHourly", message: "Day of week & Hourly" });
        },
      },
      {
        id: "signed-bars",
        get label() {
          return localize({ id: "reports.winLossBars", message: "Win / loss bars" });
        },
      },
      {
        id: "duration-scatter",
        get label() {
          return localize({ id: "reports.durationScatter", message: "Duration scatter" });
        },
      },
      {
        id: "pnl-heatmap",
        get label() {
          return localize({ id: "reports.pLHeatmap", message: "P&L heatmap" });
        },
      },
      {
        id: "sessions",
        get label() {
          return localize({ id: "reports.sessionPerformance2", message: "Session performance" });
        },
      },
      {
        id: "symbol-heatmap",
        get label() {
          return localize({ id: "reports.symbolHeatmap", message: "Symbol heatmap" });
        },
      },
    ];
  },
  get risk() {
    return [
      {
        id: "drawdown",
        get label() {
          return localize({ id: "reports.riskDrawdown2", message: "Risk & drawdown" });
        },
      },
      {
        id: "monte-carlo",
        get label() {
          return localize({
            id: "reports.monteCarloSimulation",
            message: "Monte Carlo simulation",
          });
        },
      },
      {
        id: "rule-compliance",
        get label() {
          return localize({ id: "reports.ruleCompliance", message: "Rule compliance" });
        },
      },
    ];
  },
  get behavior() {
    return [
      {
        id: "revenge",
        get label() {
          return localize({ id: "reports.revengeTrading", message: "Revenge trading" });
        },
      },
      {
        id: "overconfidence",
        get label() {
          return localize({ id: "reports.overconfidence", message: "Overconfidence" });
        },
      },
      {
        id: "loss-aversion",
        get label() {
          return localize({ id: "reports.lossAversion", message: "Loss aversion" });
        },
      },
    ];
  },
};

export const REPORT_TAB_IDS = Object.keys(REPORT_CARDS) as ReportsTab[];

export function defaultCardIds(tab: ReportsTab): string[] {
  return REPORT_CARDS[tab].map((c) => c.id);
}

/**
 * Keep only ids this tab knows (order preserved, duplicates dropped). An
 * empty or malformed list falls back to the defaults — a preset saved by a
 * newer client, or a hand-edited blob, degrades to "show everything" rather
 * than a blank tab.
 */
export function sanitizeCardIds(tab: ReportsTab, ids: unknown): string[] {
  if (!Array.isArray(ids)) return defaultCardIds(tab);
  const known = new Set(defaultCardIds(tab));
  const out: string[] = [];
  for (const id of ids) {
    if (typeof id !== "string" || !known.has(id) || out.includes(id)) continue;
    out.push(id);
  }
  return out.length > 0 ? out : defaultCardIds(tab);
}
