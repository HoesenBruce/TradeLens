import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import { analyticsApi } from "@/lib/api/analytics";
import { marketApi } from "@/lib/api/market";
import type { Account, AccountValue } from "@/lib/api/types";
import { useDisplayPrefs } from "@/lib/displayPrefs";
import { useAccountValue } from "./useAnalytics";

vi.mock("@/lib/api/analytics", () => ({
  analyticsApi: { accountValue: vi.fn<typeof analyticsApi.accountValue>() },
}));
vi.mock("@/lib/api/market", () => ({ marketApi: { fx: vi.fn<typeof marketApi.fx>() } }));
const accounts = [
  { id: "sbi", base_currency: "JPY", account_type: "cash" },
  { id: "ibkr", base_currency: "USD", account_type: "margin" },
  { id: "paper", base_currency: "JPY", account_type: "backtest" },
] as Account[];
function value(currency: string, amount: number): AccountValue {
  return {
    currency,
    timezone: "Asia/Tokyo",
    adjustment_status: "unadjusted",
    points: [
      {
        date: "2025-05-19",
        contributed_capital: amount,
        estimated_account_value: amount,
        cash_balance: amount,
        open_position_value: 0,
        realized_pnl: 0,
        unrealized_pnl: 0,
        status: "complete",
        warnings: [],
      },
    ],
  };
}
function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  useDisplayPrefs.setState({ displayCurrency: "JPY" });
  vi.mocked(analyticsApi.accountValue).mockImplementation(async (f) =>
    f.account_id === "ibkr" ? value("USD", 10_000) : value("JPY", 1_350_000),
  );
  vi.mocked(marketApi.fx).mockResolvedValue({
    from: "USD",
    to: "JPY",
    rate: 157,
    as_of: "2025-05-19",
    provider: "test",
    cached: false,
  });
});

describe("useAccountValue", () => {
  it("uses API source currency for All Accounts and never converts JPY to JPY", async () => {
    const { result } = renderHook(() => useAccountValue({}, [accounts[0]]), {
      wrapper: makeWrapper(),
    });
    await waitFor(() => expect(result.current.data?.points[0].contributed_capital).toBe(1_350_000));
    expect(result.current.data?.currency).toBe("JPY");
    expect(marketApi.fx).not.toHaveBeenCalled();
    expect(analyticsApi.accountValue).toHaveBeenCalledWith({ account_id: "sbi" });
  });

  it("converts mixed All Accounts and keeps range/ignored prices on each request", async () => {
    const filters = { from: "2025-05-19", ignored_missing_prices: "1306:2025-05-19" };
    const { result } = renderHook(() => useAccountValue(filters, accounts), {
      wrapper: makeWrapper(),
    });
    await waitFor(() => expect(result.current.data?.points[0].contributed_capital).toBe(2_920_000));
    expect(analyticsApi.accountValue).toHaveBeenCalledTimes(2);
    expect(analyticsApi.accountValue).toHaveBeenCalledWith({ ...filters, account_id: "ibkr" });
    expect(marketApi.fx).toHaveBeenCalledExactlyOnceWith({ from: "USD", to: "JPY" });
  });

  it("switches between single account and All Accounts without stale totals", async () => {
    const { result, rerender } = renderHook(
      ({ id }) => useAccountValue({ account_id: id }, accounts),
      {
        wrapper: makeWrapper(),
        initialProps: { id: undefined as string | undefined },
      },
    );
    await waitFor(() => expect(result.current.data?.points[0].contributed_capital).toBe(2_920_000));
    rerender({ id: "sbi" });
    await waitFor(() => expect(result.current.data?.points[0].contributed_capital).toBe(1_350_000));
    rerender({ id: "ibkr" });
    await waitFor(() => expect(result.current.data?.points[0].contributed_capital).toBe(1_570_000));
    rerender({ id: undefined });
    await waitFor(() => expect(result.current.data?.points[0].contributed_capital).toBe(2_920_000));
  });

  it("handles an explicit mixed-currency selection and waits for FX before exposing totals", async () => {
    let resolveFx!: (rate: Awaited<ReturnType<typeof marketApi.fx>>) => void;
    vi.mocked(marketApi.fx).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveFx = resolve;
        }),
    );
    const { result } = renderHook(() => useAccountValue({ account_id: "sbi,ibkr" }, accounts), {
      wrapper: makeWrapper(),
    });
    await waitFor(() => expect(marketApi.fx).toHaveBeenCalled());
    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
    resolveFx({
      from: "USD",
      to: "JPY",
      rate: 157,
      as_of: "2025-05-19",
      provider: "test",
      cached: false,
    });
    await waitFor(() =>
      expect(result.current.data?.points[0].estimated_account_value).toBe(2_920_000),
    );
  });

  it("does not publish a mixed total with an invalid FX rate", async () => {
    vi.mocked(marketApi.fx).mockResolvedValue({
      from: "USD",
      to: "JPY",
      rate: 0,
      as_of: "2025-05-19",
      provider: "test",
      cached: false,
    });
    const { result } = renderHook(() => useAccountValue({}, accounts), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.data).toBeUndefined();
  });

  it("follows response currency even when account metadata disagrees", async () => {
    useDisplayPrefs.setState({ displayCurrency: null });
    const { result } = renderHook(
      () => useAccountValue({}, [{ ...accounts[0], base_currency: "USD" }]),
      { wrapper: makeWrapper() },
    );
    await waitFor(() => expect(result.current.data?.currency).toBe("JPY"));
    expect(result.current.data?.points[0].estimated_account_value).toBe(1_350_000);
  });

  it("shows no partial aggregate while an account request fails", async () => {
    vi.mocked(analyticsApi.accountValue).mockRejectedValue(new Error("unavailable"));
    const { result } = renderHook(() => useAccountValue({}, accounts), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.data).toBeUndefined();
  });
  it("preserves the USD target for mixed historical series without a display override", async () => {
    useDisplayPrefs.setState({ displayCurrency: null });
    vi.mocked(marketApi.fx).mockImplementation(async ({ from, to }) => ({
      from,
      to,
      rate: 1 / 150,
      as_of: "2025-05-19",
      provider: "test",
      cached: false,
    }));
    const { result } = renderHook(() => useAccountValue({}, accounts), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.data?.currency).toBe("USD"));
    expect(result.current.data?.points[0].estimated_account_value).toBe(19000);
    expect(marketApi.fx).toHaveBeenCalledWith({ from: "JPY", to: "USD" });
  });
});
