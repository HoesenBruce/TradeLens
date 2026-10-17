import { useState } from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vite-plus/test";
import { render } from "@/test/render";
import type { TradeFilter } from "@/lib/tradeFilters";
import { TradesFilters } from "./TradesFilters";

function ControlledFilters() {
  const [conditions, setConditions] = useState<TradeFilter[]>([]);
  return (
    <>
      <TradesFilters conditions={conditions} onConditionsChange={setConditions} />
      <output aria-label="Applied conditions">{JSON.stringify(conditions)}</output>
    </>
  );
}

describe("TradesFilters", () => {
  it("keeps an added multiselect open, applies both values, and removes the condition accessibly", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(<ControlledFilters />);
    await user.click(screen.getByRole("button", { name: "Add filter" }));
    const search = await screen.findByPlaceholderText("Search filters…");
    await user.type(search, "Direction");
    await user.keyboard("{ArrowDown}{ArrowRight}");
    await user.click(await screen.findByRole("option", { name: "Long" }));
    await user.click(await screen.findByRole("option", { name: "Short" }));
    await waitFor(() =>
      expect(screen.getByLabelText("Applied conditions")).toHaveTextContent(
        '"values":["long","short"]',
      ),
    );
    await user.keyboard("{Escape}{Escape}");
    await user.click(screen.getByRole("button", { name: "Add filter" }));
    await waitFor(() =>
      expect(screen.queryByRole("option", { name: "Direction" })).not.toBeInTheDocument(),
    );
    await user.keyboard("{Escape}");
    await user.click(screen.getByRole("button", { name: "Clear Direction filter" }));
    expect(screen.getByLabelText("Applied conditions")).toHaveTextContent("[]");
  });

  it("keeps operator changes controlled and reads the selected values back", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(<ControlledFilters />);
    await user.click(screen.getByRole("button", { name: "Add filter" }));
    await user.type(await screen.findByPlaceholderText("Search filters…"), "Direction");
    await user.keyboard("{ArrowDown}{ArrowRight}");
    await user.click(await screen.findByRole("option", { name: "Short" }));
    await user.keyboard("{Escape}{Escape}");
    await user.click(screen.getByRole("button", { name: "Include" }));
    await user.click(await screen.findByRole("menuitem", { name: "Exclude" }));
    expect(screen.getByLabelText("Applied conditions")).toHaveTextContent(
      '"operator":"is_not_any_of"',
    );
    await user.click(screen.getByRole("button", { name: "Short" }));
    expect(await screen.findByRole("option", { name: "Short" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
  });
});
