import { afterEach, expect, it } from "vite-plus/test";
import { screen, cleanup } from "@testing-library/react";
import { render } from "@/test/render";
import { loadLocale } from "@/i18n";
import type { TradeQuantities } from "@/lib/api/types";
import { TradeQuantityCell, TradeQuantityDetails } from "./TradeQuantities";

afterEach(async () => {
  cleanup();
  await loadLocale("en");
});

it.each([
  [500, 400, 100, "partially_closed"],
  [700, 400, 300, "partially_closed"],
  [500, 500, 0, "closed"],
  [70, 60, 10, "partially_closed"],
  [200, 0, 200, "open"],
  [0.123456789, 0.023456789, 0.1, "partially_closed"],
] as const)("renders inventory %s/%s/%s consistently", (entry, exit, open, lifecycle) => {
  const quantities: TradeQuantities = {
    entry_qty: entry,
    exit_qty: exit,
    open_qty: open,
    remaining_pct: (open / entry) * 100,
    lifecycle,
  };
  render(
    <>
      <TradeQuantityCell trade={{ quantities }} />
      <TradeQuantityDetails trade={{ quantities }} />
    </>,
  );
  for (const bar of screen.getAllByRole("progressbar")) {
    expect(Number(bar.getAttribute("aria-valuenow"))).toBeCloseTo((exit / entry) * 100);
    expect(bar).toHaveAttribute("aria-valuemin", "0");
    expect(bar).toHaveAttribute("aria-valuemax", "100");
    expect(bar).toHaveAttribute("aria-valuetext", expect.stringContaining(`Remaining ${open}`));
  }
  for (const [label, value] of [
    ["Entry Qty", entry],
    ["Exit Qty", exit],
    ["Open Qty", open],
  ]) {
    expect(screen.getByText(String(label)).parentElement).toHaveTextContent(String(value));
  }
});

it("keeps unknown inventory explicit, without presenting an invented zero-percent bar", () => {
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
  expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  expect(
    screen.getByText("Quantity is unknown because opening history is incomplete."),
  ).toBeVisible();
  expect(screen.getByText("Position unknown")).toBeVisible();
});

it("does not reconstruct quantities on a client connected to an older server", () => {
  render(<TradeQuantityCell trade={{}} />);
  expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  expect(screen.getByText("Quantity details are unavailable from this server.")).toBeVisible();
});

it("shows full long and fractional quantities in Chinese and Japanese", async () => {
  const quantities: TradeQuantities = {
    entry_qty: 1234567890123.125,
    exit_qty: 1234567890000,
    open_qty: 123.125,
    remaining_pct: 0.00000001,
    lifecycle: "partially_closed",
  };
  for (const locale of ["en", "zh-CN", "ja"]) {
    await loadLocale(locale);
    render(<TradeQuantityDetails trade={{ quantities }} />);
    expect(screen.getByText("1,234,567,890,123.125")).toBeVisible();
    expect(screen.getByText("123.125")).toBeVisible();
    expect(Number(screen.getByRole("progressbar").getAttribute("aria-valuenow"))).toBeLessThan(100);
    expect(screen.getByText("<0.1%")).toBeVisible();
    cleanup();
  }
});
