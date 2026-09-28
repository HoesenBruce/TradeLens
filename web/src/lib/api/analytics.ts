import { apiFetch, qs } from "./client";
import type {
  AccountValue,
  BehaviorReport,
  BreakGroup,
  ComplianceReport,
  EquityCurve,
  DailyPnl,
  ExecScoreReport,
  Filters,
  MonteCarloResult,
  RSummary,
  Summary,
} from "./types";

export const analyticsApi = {
  summary: (f: Filters & { target_currency?: string }) =>
    apiFetch<Summary>(`/analytics/summary${qs(f as Record<string, string | undefined>)}`),
  rSummary: (f: Filters) =>
    apiFetch<RSummary>(`/analytics/r-summary${qs(f as Record<string, string | undefined>)}`),
  equityCurve: (f: Filters & { target_currency?: string }) =>
    apiFetch<EquityCurve>(`/analytics/equity-curve${qs(f as Record<string, string | undefined>)}`),
  accountValue: (
    f: Pick<Filters, "account_id" | "from" | "to"> & { ignored_missing_prices?: string },
  ) =>
    apiFetch<AccountValue>(
      `/analytics/account-value${qs(f as Record<string, string | undefined>)}`,
    ),
  daily: (f: Filters & { target_currency?: string }) =>
    apiFetch<DailyPnl>(`/analytics/daily${qs(f as Record<string, string | undefined>)}`),
  compliance: (f: Filters) =>
    apiFetch<ComplianceReport>(
      `/analytics/compliance${qs(f as Record<string, string | undefined>)}`,
    ),
  behavior: (f: Filters) =>
    apiFetch<BehaviorReport>(`/analytics/behavior${qs(f as Record<string, string | undefined>)}`),
  breakdown: (by: string, f: Filters) =>
    apiFetch<BreakGroup[]>(
      `/analytics/breakdown${qs({ by, ...(f as Record<string, string | undefined>) })}`,
    ),
  monteCarlo: (f: Filters) =>
    apiFetch<MonteCarloResult>(
      `/analytics/montecarlo${qs(f as Record<string, string | undefined>)}`,
    ),
  executionScore: (f: Filters, bucket: "week" | "month") =>
    apiFetch<ExecScoreReport>(
      `/analytics/execution-score${qs({ bucket, ...(f as Record<string, string | undefined>) })}`,
    ),
};
