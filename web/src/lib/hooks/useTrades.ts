import { useQuery } from "@tanstack/react-query";
import { tradesApi } from "@/lib/api/trades";
import { useAnalyticsRequest } from "./useAnalytics";
import type { Filters } from "@/lib/api/types";

export function useTrades(filters: Filters) {
  const { request, accountsQ } = useAnalyticsRequest(filters, true, true);
  const query = useQuery({
    queryKey: ["trades", request],
    queryFn: () => tradesApi.list(request),
    enabled: accountsQ.isSuccess,
  });
  return {
    ...query,
    data: query.data?.trades,
    currency: query.data?.currency,
    isLoading: accountsQ.isPending || query.isLoading,
    isError: accountsQ.isError || query.isError,
  };
}
