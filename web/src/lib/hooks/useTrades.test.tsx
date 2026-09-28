import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import { useTrades } from "./useTrades";

import { accountsApi } from "@/lib/api/accounts";
import { tradesApi } from "@/lib/api/trades";
import type { Account } from "@/lib/api/types";
import { useDisplayPrefs } from "@/lib/displayPrefs";
vi.mock("@/lib/api/accounts", () => ({ accountsApi: { list: vi.fn<typeof accountsApi.list>() } }));
vi.mock("@/lib/api/trades", () => ({ tradesApi: { list: vi.fn<typeof tradesApi.list>() } }));
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(accountsApi.list).mockResolvedValue([
    { id: "a1", base_currency: "JPY", account_type: "cash" },
    { id: "a2", base_currency: "USD", account_type: "cash" },
  ] as Account[]);
  useDisplayPrefs.setState({ displayCurrency: "JPY" });
});

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useTrades", () => {
  it("fetches the trade list for the current filters", async () => {
    vi.mocked(tradesApi.list).mockResolvedValue({
      currency: "JPY",
      target_currency: "JPY",
      fx_policy: "latest",
      fx_rates: [],
      trades: [{ id: "t1", symbol: "AAPL", net_pnl: 198 }],
    } as unknown as Awaited<ReturnType<typeof tradesApi.list>>);
    const { result } = renderHook(() => useTrades({ account_id: "a1" }), {
      wrapper,
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].symbol).toBe("AAPL");
    expect(tradesApi.list).toHaveBeenCalledWith({ account_id: "a1", target_currency: "JPY" });
  });
  it("refetches mixed All Accounts on display currency changes and fails closed", async () => {
    vi.mocked(tradesApi.list).mockImplementation(async (f) => {
      if (f.target_currency === "USD") throw new Error("fx_unavailable");
      return {
        currency: "JPY",
        target_currency: "JPY",
        fx_policy: "latest",
        fx_rates: [],
        trades: [],
      } as Awaited<ReturnType<typeof tradesApi.list>>;
    });
    const { result } = renderHook(() => useTrades({}), { wrapper });
    await waitFor(() => expect(result.current.currency).toBe("JPY"));
    await act(() => useDisplayPrefs.setState({ displayCurrency: "USD" }));
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.data).toBeUndefined();
    expect(tradesApi.list).toHaveBeenLastCalledWith({ target_currency: "USD" });
    await act(() => useDisplayPrefs.setState({ displayCurrency: "JPY" }));
    await waitFor(() => expect(result.current.currency).toBe("JPY"));
  });
});
