import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Filters } from "@/lib/api/types";
import { useAnalyticsRequest } from "./useAnalytics";
import { type AnnualGoal, settingsApi } from "@/lib/api/settings";

export function useAnnualGoal(year: number, filters: Filters = {}) {
  const { request, accountsQ } = useAnalyticsRequest(filters, true, true);
  const currency = "target_currency" in request ? request.target_currency : undefined;
  const query = useQuery({
    queryKey: ["settings", "annual-goal", year, currency],
    queryFn: () => settingsApi.getAnnualGoal(year, currency),
    enabled: accountsQ.isSuccess,
  });
  return {
    ...query,
    data: query.isError ? undefined : query.data,
    currency: currency,
    isLoading: accountsQ.isPending || query.isLoading,
    isError: accountsQ.isError || query.isError,
  };
}

export function useSaveAnnualGoal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { year: number; amount: number; currency?: string }) =>
      settingsApi.putAnnualGoal(body),
    onSuccess: (data: AnnualGoal) => {
      void qc.invalidateQueries({ queryKey: ["settings", "annual-goal", data.year] });
    },
  });
}

export function useClearAnnualGoal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (year: number) => settingsApi.deleteAnnualGoal(year),
    onSuccess: (data: AnnualGoal) => {
      void qc.invalidateQueries({ queryKey: ["settings", "annual-goal", data.year] });
    },
  });
}
