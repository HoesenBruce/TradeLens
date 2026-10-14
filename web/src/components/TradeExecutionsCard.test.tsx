import { expect, it } from "vite-plus/test";
import { screen, cleanup } from "@testing-library/react";
import { render } from "@/test/render";
import type { Execution, TradeDetail } from "@/lib/api/types";
import { TradeExecutionsCard } from "./TradeExecutionsCard";

it("shows date-only source records without inventing running positions", () => {
  const fills = ["sell", "buy", "buy", "sell"].map(
    (side, i) =>
      ({
        id: String(i),
        user_id: "synthetic-user",
        account_id: "synthetic-account",
        external_id: null,
        symbol: "TEST",
        instrument_type: "stock",
        import_batch_id: null,
        dedup_hash: String(i),
        created_at: "2026-09-01T00:00:00Z",
        side,
        quantity: 10,
        price: 100,
        fees: 0,
        commission: 0,
        multiplier: 1,
        executed_at: `2026-09-01T00:00:00.00000${i}Z`,
        details: { lot: "sbi:margin-long", source_time_precision: "date" },
      }) as Execution,
  );
  const trade = { direction: "long", pnl_currency: "JPY", fills } as TradeDetail;
  render(<TradeExecutionsCard trade={trade} />);
  expect(screen.queryByRole("columnheader", { name: "Pos" })).not.toBeInTheDocument();
  expect(screen.queryByRole("columnheader", { name: "Time" })).not.toBeInTheDocument();
  expect(screen.getByRole("columnheader", { name: "Executed" })).toBeInTheDocument();
  expect(screen.getByText("Sep 1, 2026")).toBeInTheDocument();
  cleanup();
  render(
    <TradeExecutionsCard
      trade={{ ...trade, fills: fills.map((f) => ({ ...f, details: null })) }}
    />,
  );
  expect(screen.getByRole("columnheader", { name: "Pos" })).toBeInTheDocument();
  expect(screen.getByRole("columnheader", { name: "Time" })).toBeInTheDocument();
});
