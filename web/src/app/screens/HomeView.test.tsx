import { PRIVACY_MASK, useDisplayPrefs } from "@/lib/displayPrefs";
import { render } from "@/test/render";
import type { ColumnDef } from "@/lib/table";
import { flexRender, getCoreRowModel, useReactTable, type RowData } from "@/lib/table";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vite-plus/test";
import type { AccountValueWarning, Summary, Trade } from "@/lib/api/types";
import { AccountValueTooltip, HomeView } from "./HomeView";

vi.mock("../../components/Toast", () => ({
  useToastManager: () => ({ add: vi.fn<(...args: any[]) => any>() }),
}));

vi.mock("../../lib/hooks/useTradeDetail", () => ({
  useDeleteTrade: () => ({
    mutateAsync: vi.fn<(...args: any[]) => any>(),
    isPending: false,
  }),
}));

vi.mock("../../lib/hooks/useMoneyFx", () => ({
  useMoneyFx: (baseCurrency: string) => ({
    baseCurrency,
    displayCurrency: baseCurrency || "USD",
    currency: baseCurrency || "USD",
    rate: 1,
    toDisplay: (v: number) => v,
    isLoading: false,
    isError: false,
  }),
}));

// DailyLossCard fetches risk rules; no limit configured means it renders null.
vi.mock("../../lib/hooks/useRiskRules", () => ({
  useRiskRules: () => ({ data: undefined, isLoading: false, isError: false }),
}));

// PropStatusCard fetches prop status; unconfigured means it renders null.
vi.mock("../../lib/hooks/useProp", () => ({
  usePropStatus: () => ({ data: undefined, isLoading: false, isError: false }),
}));

// Mock DataTable: the real one uses a virtualizer that needs a sized container
// (absent in jsdom, so it renders zero rows). Mirrors
// src/components/tradeColumns.test.tsx, which hits the same jsdom gotcha.
vi.mock("../../components/DataTable", () => ({
  DataTable: function MockDataTable<T extends RowData>({
    columns,
    data,
  }: {
    columns: ColumnDef<T>[];
    data: T[];
  }) {
    const table = useReactTable({
      data,
      columns,
      getCoreRowModel: getCoreRowModel(),
    });

    return (
      <table>
        <thead>
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    );
  },
}));

const SUMMARY: Summary = {
  total_trades: 4,
  wins: 2,
  losses: 2,
  breakeven: 0,
  win_rate: 0.5,
  net_pnl: -61.79,
  // The API reports loss figures as positive magnitudes (analytics.Summarize).
  gross_profit: 69.48,
  gross_loss: 131.27,
  profit_factor: 0.53,
  expectancy: -15.45,
  avg_win: 34.74,
  avg_loss: 65.64,
  avg_trade: -15.45,
  largest_win: 58.09,
  largest_loss: 111.24,
  total_fees: 12.5,
};

const TRADE: Trade = {
  id: "t1",
  account_id: "a1",
  symbol: "TSLQ",
  instrument_type: "stock",
  direction: "long",
  status: "closed",
  opened_at: "2026-07-02T13:00:00Z",
  closed_at: "2026-07-02T13:39:00Z",
  qty_opened: 80,
  qty_remaining: 0,
  avg_entry_price: 18.02,
  avg_exit_price: 18.23,
  gross_pnl: 16.8,
  fees_total: 5.41,
  net_pnl: 11.39,
  pnl_currency: "USD",
  return_pct: 0.79,
  time_in_trade_secs: 2340,
  notes: "",
  tags: [],
};

const BASE = {
  baselineTrades: [TRADE],
  summaryLoading: false,
  summaryError: false,
  summary: SUMMARY,
  equityLoading: false,
  equityError: false,
  equityPoints: [
    { at: "2026-07-01T00:00:00Z", equity: -20.03 },
    { at: "2026-07-02T00:00:00Z", equity: -61.79 },
  ],
  accountValue: {
    currency: "USD",
    timezone: "Asia/Tokyo",
    adjustment_status: "unadjusted",
    points: [
      {
        date: "2026-07-01",
        estimated_account_value: 1000,
        contributed_capital: 900,
        cash_balance: 800,
        open_position_value: 200,
        realized_pnl: 0,
        unrealized_pnl: 100,
        status: "complete",
        warnings: [],
      },
    ],
  },
  accountValueLoading: false,
  accountValueError: false,
  accountValueRange: "30D",
  onAccountValueRangeChange: vi.fn<(...args: any[]) => any>(),
  onIgnoreAccountValueWarning: vi.fn<(...args: any[]) => any>(),
  tradesLoading: false,
  tradesError: false,
  trades: [TRADE],
  accounts: [],
  selectedAccountIds: undefined,
  onSelectTrade: vi.fn<(...args: any[]) => any>(),
  onOpenFullPage: vi.fn<(...args: any[]) => any>(),
  onViewAllTrades: vi.fn<(...args: any[]) => any>(),
  onOpenCalendar: vi.fn<(...args: any[]) => any>(),
  onOpenReports: vi.fn<(...args: any[]) => any>(),
  calendarYear: 2026,
  calendarMonth: 7,
  dailyPnl: { "2026-07-02": 11.39 },
  dailyLoading: false,
  dailyError: false,
  breakdownDim: "day_of_week" as const,
  onBreakdownDimChange: vi.fn<(...args: any[]) => any>(),
  breakdown: [
    {
      key: "Wed",
      summary: {
        ...SUMMARY,
        net_pnl: 11.39,
      },
    },
  ],
  breakdownLoading: false,
  breakdownError: false,
  accountFunded: false,
  onImport: vi.fn<(...args: any[]) => any>(),
  onNewTrade: vi.fn<(...args: any[]) => any>(),
  goalYear: 2026,
  goalAmount: null,
  goalLoading: false,
  goalSaving: false,
  ytdNetPnl: undefined,
  ytdLoading: false,
  onSaveGoal: vi.fn<(...args: any[]) => any>(async () => {}),
  onClearGoal: vi.fn<(...args: any[]) => any>(async () => {}),
};

describe("HomeView", () => {
  it("keeps Open statistics on the base scope while outcome-filtered content changes", () => {
    const open = { ...TRADE, id: "open", symbol: "OPEN", status: "open" as const };
    const baselineTrades = [TRADE, open];
    const { rerender } = render(
      <HomeView {...BASE} baselineTrades={baselineTrades} trades={baselineTrades} />,
    );
    for (const filter of ["open", "win", "loss", "wash", undefined] as const) {
      const trades =
        filter === "open" ? [open] : filter === "win" ? [TRADE] : filter ? [] : baselineTrades;
      rerender(
        <HomeView
          {...BASE}
          baselineTrades={baselineTrades}
          trades={trades}
          tradeStatusFilter={filter}
        />,
      );
      const card = screen.getByRole("button", { name: /^Open 1 50%$/ });
      expect(within(card).getByText("1")).toBeInTheDocument();
      expect(card).toHaveAttribute("aria-pressed", String(filter === "open"));
    }
    rerender(
      <HomeView
        {...BASE}
        summary={{ ...SUMMARY, total_trades: 0 }}
        baselineTrades={[open]}
        trades={[]}
        tradeStatusFilter="loss"
      />,
    );
    expect(screen.getByRole("button", { name: /^Open 1 100%$/ })).toBeInTheDocument();
  });

  it.each([
    [1000, 900, "+$100.00"],
    [800, 900, "-$100.00"],
    [900, 900, "$0.00"],
  ])("shows account value tooltip gain/loss for %i versus %i", (value, capital, gainLoss) => {
    render(
      <AccountValueTooltip
        point={{ estimated_account_value: value, contributed_capital: capital }}
        currency="USD"
        label={Date.UTC(2026, 6, 1)}
      />,
    );
    expect(screen.getByText(/Net Contributions:/)).toHaveTextContent("$900.00");
    expect(screen.getByText(/Account Value:/)).toHaveTextContent(
      new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(value),
    );
    expect(screen.getByText(/Gain \/ Loss:/)).toHaveTextContent(gainLoss);
    expect(screen.queryByText(/Contributed Capital:/)).not.toBeInTheDocument();
  });

  it("remasks account value tooltip while mounted and across currency changes", () => {
    const point = { estimated_account_value: 1000, contributed_capital: 900 };
    const { rerender } = render(
      <AccountValueTooltip point={point} currency="USD" label={Date.UTC(2026, 9, 8)} />,
    );
    expect(screen.getByText(/Account Value:/)).toHaveTextContent("$1,000.00");
    act(() => useDisplayPrefs.getState().setPrivacyMode(true));
    expect(screen.getAllByText(new RegExp(PRIVACY_MASK))).toHaveLength(3);
    rerender(<AccountValueTooltip point={point} currency="JPY" label={Date.UTC(2026, 9, 8)} />);
    expect(screen.getAllByText(new RegExp(PRIVACY_MASK))).toHaveLength(3);
    act(() => useDisplayPrefs.getState().setPrivacyMode(false));
    expect(screen.getByText(/Account Value:/)).toHaveTextContent("¥1,000");
  });

  it("omits account value and gain/loss when reconstruction is unavailable", () => {
    render(
      <AccountValueTooltip
        point={{ estimated_account_value: null, contributed_capital: 900 }}
        currency="USD"
        label={Date.UTC(2026, 6, 1)}
      />,
    );
    expect(screen.getByText(/Net Contributions:/)).toBeInTheDocument();
    expect(screen.queryByText(/Account Value:/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Gain \/ Loss:/)).not.toBeInTheDocument();
  });

  it("renders the stats strip from the summary", () => {
    render(<HomeView {...BASE} />);
    expect(screen.getByText("Wins")).toBeInTheDocument();
    expect(screen.getByText("Losses")).toBeInTheDocument();
    expect(screen.getByText("Avg win")).toBeInTheDocument();
    expect(screen.getByText("Avg loss")).toBeInTheDocument();
    expect(screen.getByText(/Gross/)).toBeInTheDocument();
    expect(screen.getByText(/^Net$/)).toBeInTheDocument();
    expect(screen.getByText(/PF/)).toBeInTheDocument();
    // Gross must reconcile: gross - fees = net (-49.29 - 12.50 = -61.79), never the
    // sum of the win/loss buckets (+200.75).
    expect(screen.getByText("-$49.29")).toBeInTheDocument();
    expect(screen.getAllByText("0.53").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/-\$61\.79/).length).toBeGreaterThan(0);
  });

  it("renders recent trades with view-all action", async () => {
    const user = userEvent.setup();
    const onViewAllTrades = vi.fn<(...args: any[]) => any>();
    render(<HomeView {...BASE} onViewAllTrades={onViewAllTrades} />);
    expect(screen.getByText("Recent trades")).toBeInTheDocument();
    expect(screen.getAllByText("TSLQ").length).toBeGreaterThan(0);
    expect(screen.getByText("WIN")).toBeInTheDocument();
    expect(screen.getByText("1 trade")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /view all trades/i }));
    expect(onViewAllTrades).toHaveBeenCalledOnce();
  });

  it("renders insight bento panels", () => {
    render(<HomeView {...BASE} />);
    expect(screen.getByText("PnL quality")).toBeInTheDocument();
    expect(screen.getByText("Stability")).toBeInTheDocument();
    expect(screen.getByText("Time")).toBeInTheDocument();
    expect(screen.getByText("Expectancy")).toBeInTheDocument();
    expect(screen.getByText("Best streak")).toBeInTheDocument();
    expect(screen.getByText("Average hold")).toBeInTheDocument();
  });

  it("renders breakdown chart and mini calendar", async () => {
    const user = userEvent.setup();
    const onOpenCalendar = vi.fn<(...args: any[]) => any>();
    const onOpenReports = vi.fn<(...args: any[]) => any>();
    render(<HomeView {...BASE} onOpenCalendar={onOpenCalendar} onOpenReports={onOpenReports} />);
    expect(screen.getByText("Breakdown")).toBeInTheDocument();
    expect(screen.getByText("Month")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /full calendar/i })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /full calendar/i }));
    expect(onOpenCalendar).toHaveBeenCalledOnce();
    await user.click(screen.getByRole("button", { name: /reports/i }));
    expect(onOpenReports).toHaveBeenCalledOnce();
  });

  it("shows account contribution when multiple accounts have trades", () => {
    render(
      <HomeView
        {...BASE}
        accounts={[
          {
            id: "a1",
            user_id: "u",
            name: "Prop",
            broker: "x",
            account_type: "live",
            base_currency: "USD",
            starting_balance: 0,
            created_at: "2026-01-01",
          },
          {
            id: "a2",
            user_id: "u",
            name: "Personal",
            broker: "x",
            account_type: "live",
            base_currency: "USD",
            starting_balance: 0,
            created_at: "2026-01-01",
          },
        ]}
        trades={[TRADE, { ...TRADE, id: "t2", account_id: "a2", symbol: "ES", net_pnl: 50 }]}
      />,
    );
    expect(screen.getByText("Account contribution")).toBeInTheDocument();
    expect(screen.getByText("Prop")).toBeInTheDocument();
    expect(screen.getByText("Personal")).toBeInTheDocument();
  });

  it("caps recent trades and reports remaining count", () => {
    const many = Array.from({ length: 12 }, (_, i) => ({
      ...TRADE,
      id: `t${i}`,
      symbol: i === 0 ? "TSLQ" : `SYM${i}`,
    }));
    render(<HomeView {...BASE} trades={many} />);
    expect(screen.getByText("Showing 10 of 12 trades")).toBeInTheDocument();
  });

  it("renders range segmented control", () => {
    render(<HomeView {...BASE} />);
    expect(screen.getAllByRole("button", { name: "30D" })).toHaveLength(2);
    expect(screen.getAllByRole("button", { name: "ALL" })).toHaveLength(2);
  });

  it("renders reconstructed account value and contributed capital", () => {
    render(<HomeView {...BASE} />);
    expect(screen.getByText("Historical account value")).toBeInTheDocument();
    expect(screen.getByText("Estimated Account Value")).toBeInTheDocument();
    expect(screen.getByText("Contributed Capital")).toBeInTheDocument();
  });

  it("shows account-value loading, empty, error, and a single warning detail", () => {
    const { rerender } = render(<HomeView {...BASE} accountValueLoading />);
    expect(screen.queryByText("Estimated Account Value")).not.toBeInTheDocument();

    rerender(<HomeView {...BASE} accountValue={undefined} />);
    expect(screen.getByText("No account value data")).toBeInTheDocument();

    rerender(<HomeView {...BASE} accountValueError />);
    expect(screen.getByText("Failed to load account value.")).toBeInTheDocument();

    rerender(
      <HomeView
        {...BASE}
        accountValue={{
          ...BASE.accountValue,
          points: [
            {
              ...BASE.accountValue.points[0],
              estimated_account_value: null,
              status: "unsupported_corporate_action",
              warnings: [
                {
                  code: "unsupported_corporate_action",
                  instrument: "5401",
                  execution_id: "execution-1",
                  date: "2026-07-01",
                  message: "split",
                },
              ],
            },
          ],
        }}
      />,
    );
    expect(screen.getByText("Corporate action review required")).toBeInTheDocument();
    expect(screen.getByText(/5401 · 2026-07-01 · Execution execution-1/)).toBeInTheDocument();
    expect(screen.getByText(/split/)).toBeInTheDocument();
  });

  it("deduplicates and collapses multiple account-value warning details", async () => {
    const user = userEvent.setup();
    render(
      <HomeView
        {...BASE}
        accountValue={{
          ...BASE.accountValue,
          points: [
            {
              ...BASE.accountValue.points[0],
              status: "unsupported_corporate_action",
              warnings: [
                {
                  code: "unsupported_corporate_action",
                  instrument: "5401",
                  date: "2026-07-01",
                  message: "split",
                },
                {
                  code: "missing_price",
                  instrument: "6501",
                  date: "2026-07-01",
                  message: "missing",
                },
              ],
            },
            {
              ...BASE.accountValue.points[0],
              date: "2026-07-02",
              status: "unsupported_corporate_action",
              warnings: [
                {
                  code: "unsupported_corporate_action",
                  instrument: "5401",
                  date: "2026-07-02",
                  message: "split",
                },
              ],
            },
          ],
        }}
      />,
    );

    expect(screen.getByText("2 account value issues")).toBeInTheDocument();
    expect(screen.queryByText(/5401 · 2026-07-01/)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /2 account value issues/i }));
    expect(screen.getByRole("button", { name: /hide/i })).toBeInTheDocument();
    expect(screen.getByText(/5401 · 2026-07-01/)).toBeInTheDocument();
    expect(screen.getByText(/6501 · 2026-07-01/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /2 account value issues/i }));
    expect(screen.getByRole("button", { name: /show all/i })).toBeInTheDocument();
    expect(screen.queryByText(/5401 · 2026-07-01/)).not.toBeInTheDocument();
  });

  it("offers ignore for a missing price and labels a carried-forward close", async () => {
    const user = userEvent.setup();
    const onIgnore = vi.fn<(warning: AccountValueWarning) => void>();
    const warning = {
      code: "missing_price",
      instrument: "1328",
      date: "2025-10-24",
      message: "no unadjusted close is available for this market session",
    };
    const { rerender } = render(
      <HomeView
        {...BASE}
        onIgnoreAccountValueWarning={onIgnore}
        accountValue={{
          ...BASE.accountValue,
          points: [
            {
              ...BASE.accountValue.points[0],
              status: "incomplete_missing_price",
              warnings: [warning],
            },
          ],
        }}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Ignore" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Ignore missing market data?")).toBeInTheDocument();
    expect(screen.getByText(/previous available market close/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onIgnore).not.toHaveBeenCalled();
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());

    await user.click(screen.getByRole("button", { name: "Ignore" }));
    await user.click(screen.getByRole("button", { name: "Ignore and use previous close" }));
    expect(onIgnore).toHaveBeenCalledWith(warning);

    rerender(
      <HomeView
        {...BASE}
        onIgnoreAccountValueWarning={onIgnore}
        accountValue={{
          ...BASE.accountValue,
          points: [
            {
              ...BASE.accountValue.points[0],
              status: "complete",
              warnings: [
                {
                  ...warning,
                  code: "carried_forward_suspension_price",
                  message: "using the previous market close after the missing price was ignored",
                },
              ],
            },
          ],
        }}
      />,
    );

    expect(screen.getByText("Estimated using previous close")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Ignore" })).not.toBeInTheDocument();
  });

  it("computes OPEN percentage against all trades, not closed-only total", () => {
    const openTrade: Trade = {
      ...TRADE,
      id: "t2",
      status: "open",
      closed_at: null,
      net_pnl: null,
      avg_exit_price: null,
      return_pct: null,
      time_in_trade_secs: null,
    };
    // Two open trades, zero closed: the closed-only total would yield
    // 2 / max(0, 1) = 200%; against all trades it is 2 / 2 = 100%.
    render(
      <HomeView
        {...BASE}
        summary={{ ...SUMMARY, total_trades: 0, wins: 0, losses: 0 }}
        trades={[openTrade, { ...openTrade, id: "t3" }]}
        baselineTrades={[openTrade, { ...openTrade, id: "t3" }]}
      />,
    );
    expect(screen.getByText("100%")).toBeInTheDocument();
    expect(screen.queryByText("200%")).toBeNull();
  });

  it("shows the empty state with onboarding actions", () => {
    render(
      <HomeView
        {...BASE}
        summary={{ ...SUMMARY, total_trades: 0 }}
        trades={[]}
        baselineTrades={[]}
        equityPoints={[]}
      />,
    );
    expect(screen.getByText("No trades yet")).toBeInTheDocument();
    expect(
      screen.getByText(
        /Import broker history or log your first trade to start tracking performance/i,
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /import csv/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /log trade/i })).toBeInTheDocument();
  });

  it("shows funded-account hint when account has cash", () => {
    render(
      <HomeView
        {...BASE}
        summary={{ ...SUMMARY, total_trades: 0 }}
        trades={[]}
        baselineTrades={[]}
        equityPoints={[]}
        accountFunded
      />,
    );
    expect(
      screen.getByText(
        /Account funded — import history or log your first trade to see P&L light up here/i,
      ),
    ).toBeInTheDocument();
  });

  it("wires empty-state actions", async () => {
    const user = userEvent.setup();
    const onImport = vi.fn<(...args: any[]) => any>();
    const onNewTrade = vi.fn<(...args: any[]) => any>();
    render(
      <HomeView
        {...BASE}
        summary={{ ...SUMMARY, total_trades: 0 }}
        trades={[]}
        baselineTrades={[]}
        equityPoints={[]}
        onImport={onImport}
        onNewTrade={onNewTrade}
      />,
    );
    await user.click(screen.getByRole("button", { name: /import csv/i }));
    await user.click(screen.getByRole("button", { name: /log trade/i }));
    expect(onImport).toHaveBeenCalledOnce();
    expect(onNewTrade).toHaveBeenCalledOnce();
  });
});

it("updates dashboard labels when switching between Chinese and Japanese", async () => {
  const { loadLocale } = await import("@/i18n");
  const { act } = await import("@testing-library/react");
  render(<HomeView {...BASE} />);
  try {
    await act(() => loadLocale("zh-CN"));
    expect(screen.getByText("权益曲线")).toBeInTheDocument();
    expect(screen.getByText("盈亏质量")).toBeInTheDocument();
    await act(() => loadLocale("ja"));
    expect(screen.getByText("資産推移")).toBeInTheDocument();
    expect(screen.getByText("損益の質")).toBeInTheDocument();
  } finally {
    await act(() => loadLocale("en"));
  }
});
