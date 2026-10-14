import { expect, it } from "vite-plus/test";
import { screen, cleanup } from "@testing-library/react";
import { render } from "@/test/render";
import type { TradeQuantities } from "@/lib/api/types";
import { TradeQuantityCell, TradeQuantityDetails } from "./TradeQuantities";

it.each([
  [500, 400, 100, 20, "partially_closed"],
  [700, 400, 300, (300 / 700) * 100, "partially_closed"],
  [500, 500, 0, 0, "closed"],
  [70, 60, 10, (10 / 70) * 100, "partially_closed"],
  [200, 0, 200, 100, "open"],
] as const)("renders inventory %s/%s/%s consistently", (entry, exit, open, pct, lifecycle) => {
  cleanup();
  const quantities: TradeQuantities = {
    entry_qty: entry,
    exit_qty: exit,
    open_qty: open,
    remaining_pct: pct,
    lifecycle,
  };
  render(
    <>
      <TradeQuantityCell trade={{ quantities }} />
      <TradeQuantityDetails trade={{ quantities }} />
    </>,
  );
  expect(screen.getByLabelText(`Open Qty ${open} / Entry Qty ${entry}`)).toHaveTextContent(
    `${open} / ${entry}`,
  );
  for (const [label, value] of [
    ["Entry Qty", entry],
    ["Exit Qty", exit],
    ["Open Qty", open],
  ]) {
    expect(screen.getByText(label).parentElement).toHaveTextContent(String(value));
  }
  expect(screen.getByText("Remaining %").parentElement).toHaveTextContent(`${pct.toFixed(1)}%`);
});

it("renders unknown inventory as dashes with an explanation", () => {
  render(
    <TradeQuantityDetails
      trade={{
        quantities: {
          entry_qty: null,
          exit_qty: null,
          open_qty: null,
          remaining_pct: null,
          lifecycle: "unknown",
          unknown_reason: "incomplete_opening",
        },
      }}
    />,
  );
  expect(screen.getAllByText("—")).toHaveLength(4);
  expect(
    screen.getByText("Quantity is unknown because opening history is incomplete."),
  ).toBeVisible();
  expect(screen.getByText("Position unknown")).toBeVisible();
});

it("does not reconstruct quantities on a client connected to an older server", () => {
  render(<TradeQuantityCell trade={{}} />);
  expect(screen.getByText("— / —")).toHaveAttribute(
    "title",
    expect.stringContaining("unavailable from this server"),
  );
});
