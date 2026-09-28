import { useQueries, useQuery } from "@tanstack/react-query";
import { analyticsApi } from "@/lib/api/analytics";
import type { Account, Filters } from "@/lib/api/types";
import { combineAccountValues } from "@/lib/accountValue";
import { accountBaseCurrency, useDisplayCurrency } from "@/lib/displayPrefs";
import { fxRateQueryOptions } from "./useMoneyFx";

export function useSummary(filters: Filters) {
  return useQuery({
    queryKey: ["analytics", "summary", filters],
    queryFn: () => analyticsApi.summary(filters),
  });
}

export function useRSummary(filters: Filters) {
  return useQuery({
    queryKey: ["analytics", "r-summary", filters],
    queryFn: () => analyticsApi.rSummary(filters),
  });
}

export function useEquityCurve(filters: Filters) {
  return useQuery({
    queryKey: ["analytics", "equity-curve", filters],
    queryFn: () => analyticsApi.equityCurve(filters),
  });
}

export function useAccountValue(
  filters: Pick<Filters, "account_id" | "from" | "to"> & { ignored_missing_prices?: string },
  accounts: Account[],
) {
  const ids = filters.account_id?.split(",");
  const selected = accounts.filter((account) =>
    ids?.length ? ids.includes(account.id) : account.account_type !== "backtest",
  );
  const series = useQueries({
    queries: selected.map((account) => {
      const accountFilters = { ...filters, account_id: account.id };
      return {
        queryKey: ["analytics", "account-value", accountFilters],
        queryFn: () => analyticsApi.accountValue(accountFilters),
      };
    }),
  });
  // API currency is authoritative, including when account metadata has changed.
  const sources = [
    ...new Set(series.flatMap((query) => (query.data ? [query.data.currency] : []))),
  ];
  const baseCurrency =
    sources.length === 1
      ? sources[0]
      : accountBaseCurrency(
          selected,
          selected.map((account) => account.id),
        );
  const currency = useDisplayCurrency(baseCurrency);
  const fx = useQueries({
    queries: sources.map((source) => fxRateQueryOptions(source, currency)),
  });
  const rates = Object.fromEntries(
    sources.map((source, i) => [source, source === currency ? 1 : fx[i].data?.rate]),
  );
  const isLoading =
    series.some((query) => query.isPending) ||
    sources.some((source, i) => source !== currency && fx[i].isPending);
  const isError =
    series.some((query) => query.isError) ||
    sources.some(
      (source, i) =>
        source !== currency &&
        (fx[i].isError ||
          (!fx[i].isPending &&
            (rates[source] == null || !Number.isFinite(rates[source]) || rates[source]! <= 0))),
    );
  return {
    isLoading,
    isError,
    data:
      isLoading || isError
        ? undefined
        : combineAccountValues(
            series.flatMap((query) => (query.data ? [query.data] : [])),
            currency,
            rates,
          ),
  };
}

export function useDailyPnl(filters: Filters) {
  return useQuery({
    queryKey: ["analytics", "daily", filters],
    queryFn: () => analyticsApi.daily(filters),
  });
}

export function useCompliance(filters: Filters, enabled = true) {
  return useQuery({
    queryKey: ["analytics", "compliance", filters],
    queryFn: () => analyticsApi.compliance(filters),
    enabled,
  });
}

export function useBehavior(filters: Filters, enabled = true) {
  return useQuery({
    queryKey: ["analytics", "behavior", filters],
    queryFn: () => analyticsApi.behavior(filters),
    enabled,
  });
}

export function useMonteCarlo(filters: Filters, enabled = true) {
  return useQuery({
    queryKey: ["analytics", "montecarlo", filters],
    queryFn: () => analyticsApi.monteCarlo(filters),
    enabled,
  });
}

export function useBreakdown(by: string, filters: Filters) {
  return useQuery({
    queryKey: ["analytics", "breakdown", by, filters],
    queryFn: () => analyticsApi.breakdown(by, filters),
  });
}

export function useExecutionScore(filters: Filters, bucket: "week" | "month") {
  return useQuery({
    queryKey: ["analytics", "execution-score", bucket, filters],
    queryFn: () => analyticsApi.executionScore(filters, bucket),
  });
}
