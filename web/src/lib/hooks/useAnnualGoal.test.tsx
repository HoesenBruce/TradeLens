import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vite-plus/test";
import { accountsApi } from "@/lib/api/accounts";
import { settingsApi } from "@/lib/api/settings";
import type { Account } from "@/lib/api/types";
import { useDisplayPrefs } from "@/lib/displayPrefs";
import { useAnnualGoal } from "./useAnnualGoal";
vi.mock("@/lib/api/accounts", () => ({ accountsApi: { list: vi.fn<typeof accountsApi.list>() } }));
vi.mock("@/lib/api/settings", () => ({
  settingsApi: { getAnnualGoal: vi.fn<typeof settingsApi.getAnnualGoal>() },
}));
beforeEach(() => {
  vi.clearAllMocks();
  useDisplayPrefs.setState({ displayCurrency: "JPY" });
  vi.mocked(accountsApi.list).mockResolvedValue([
    { id: "j", base_currency: "JPY", account_type: "cash" },
    { id: "u", base_currency: "USD", account_type: "cash" },
  ] as Account[]);
});
it("requests goals in the same currency as scope analytics, separates display keys and fails closed", async () => {
  vi.mocked(settingsApi.getAnnualGoal).mockImplementation(async (year, currency) => {
    if (currency === "USD") throw new Error("FX unavailable");
    return { year: year!, amount: 150000, currency };
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { result, rerender } = renderHook(({ id }) => useAnnualGoal(2026, { account_id: id }), {
    initialProps: { id: undefined as string | undefined },
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  });
  await waitFor(() => expect(result.current.data?.currency).toBe("JPY"));
  await act(() => useDisplayPrefs.setState({ displayCurrency: "USD" }));
  await waitFor(() => expect(result.current.isError).toBe(true));
  expect(result.current.data).toBeUndefined();
  expect(settingsApi.getAnnualGoal).toHaveBeenLastCalledWith(2026, "USD");
  rerender({ id: "j" });
  await waitFor(() => expect(result.current.data?.currency).toBe("JPY"));
  expect(result.current.currency).toBe("JPY");
  vi.mocked(settingsApi.getAnnualGoal).mockRejectedValueOnce(new Error("FX unavailable"));
  await act(() => result.current.refetch());
  expect(result.current.isError).toBe(true);
  expect(result.current.data).toBeUndefined();
});
