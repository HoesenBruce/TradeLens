import { useQueries, useQuery } from "@tanstack/react-query";
import { analyticsApi } from "@/lib/api/analytics";
import type { Account, Filters } from "@/lib/api/types";
import { combineAccountValues } from "@/lib/accountValue";
import { accountBaseCurrency, useDisplayCurrency } from "@/lib/displayPrefs";
import { useAccounts } from "./useAccounts";
import { fxRateQueryOptions } from "./useMoneyFx";

export function useAnalyticsRequest(
  filters: Filters,
  normalizeMixed = true,
  explicitTarget = false,
) {
  const accountsQ = useAccounts();
  const ids = filters.account_id?.split(",").filter(Boolean);
  const currencies = new Set(
    (accountsQ.data ?? [])
      .filter((account) =>
        ids?.length ? ids.includes(account.id) : account.account_type !== "backtest",
      )
      .map((account) => account.base_currency),
  );
  // Auto has no shared base for a mixed scope: use an explicit USD target.
  // Same-currency scopes keep their native contract for forms and sibling endpoints.
  const displayCurrency = useDisplayCurrency("USD");
  const request =
    normalizeMixed && currencies.size > 1
      ? { ...filters, target_currency: displayCurrency }
      : explicitTarget
        ? {
            ...filters,
            target_currency: currencies.size === 1 ? [...currencies][0] : displayCurrency,
          }
        : filters;
  return { request, accountsQ };
}

export function useSummary(filters: Filters, normalizeMixed = true) {
  const { request, accountsQ } = useAnalyticsRequest(filters, normalizeMixed);
  const query = useQuery({
    queryKey: ["analytics", "summary", request],
    queryFn: () => analyticsApi.summary(request),
    enabled: accountsQ.isSuccess,
  });
  return {
    ...query,
    isLoading: accountsQ.isPending || query.isLoading,
    isError: accountsQ.isError || query.isError,
  };
}

export function useRSummary(filters: Filters) {
  return useQuery({
    queryKey: ["analytics", "r-summary", filters],
    queryFn: () => analyticsApi.rSummary(filters),
  });
}

export function useEquityCurve(filters: Filters) {
  const { request, accountsQ } = useAnalyticsRequest(filters, true, true);
  const query = useQuery({
    queryKey: ["analytics", "equity-curve", request],
    queryFn: () => analyticsApi.equityCurve(request),
    enabled: accountsQ.isSuccess,
  });
  return {
    ...query,
    isLoading: accountsQ.isPending || query.isLoading,
    isError: accountsQ.isError || query.isError,
  };
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
          // Historical series already convert each API source separately;
          // preserve their existing USD target default for mixed scopes.
          "USD",
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
  const { request, accountsQ } = useAnalyticsRequest(filters, true, true);
  const query = useQuery({
    queryKey: ["analytics", "daily", request],
    queryFn: () => analyticsApi.daily(request),
    enabled: accountsQ.isSuccess,
  });
  return {
    ...query,
    isLoading: accountsQ.isPending || query.isLoading,
    isError: accountsQ.isError || query.isError,
  };
}

export function useCompliance(filters: Filters, enabled = true) {
  return useQuery({
    queryKey: ["analytics", "compliance", filters],
    queryFn: () => analyticsApi.compliance(filters),
    enabled,
  });
}

export function useBehavior(filters: Filters, enabled = true) {
  const { request, accountsQ } = useAnalyticsRequest(filters, true, true);
  return useQuery({
    queryKey: ["analytics", "behavior", request],
    queryFn: () => analyticsApi.behavior(request),
    enabled: enabled && accountsQ.isSuccess,
  });
}

export function useMonteCarlo(filters: Filters, enabled = true) {
  const { request, accountsQ } = useAnalyticsRequest(filters, true, true);
  return useQuery({
    queryKey: ["analytics", "montecarlo", request],
    queryFn: () => analyticsApi.monteCarlo(request),
    enabled: enabled && accountsQ.isSuccess,
  });
}

export function useBreakdown(by: string, filters: Filters) {
  const { request, accountsQ } = useAnalyticsRequest(filters, true, true);
  const query = useQuery({
    queryKey: ["analytics", "breakdown", by, request],
    queryFn: () => analyticsApi.breakdown(by, request),
    enabled: accountsQ.isSuccess,
  });
  return {
    ...query,
    data: query.data?.groups,
    currency: query.data?.currency,
    isLoading: accountsQ.isPending || query.isLoading,
    isError: accountsQ.isError || query.isError,
  };
}

export function useExecutionScore(filters: Filters, bucket: "week" | "month") {
  return useQuery({
    queryKey: ["analytics", "execution-score", bucket, filters],
    queryFn: () => analyticsApi.executionScore(filters, bucket),
  });
}
