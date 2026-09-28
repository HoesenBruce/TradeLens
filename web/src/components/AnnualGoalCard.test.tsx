import { render } from "@/test/render";
import userEvent from "@testing-library/user-event";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vite-plus/test";
import { AnnualGoalCard } from "./AnnualGoalCard";
import { GoalProgressBar } from "./GoalProgressBar";

describe("GoalProgressBar", () => {
  it("exposes progressbar semantics", () => {
    render(<GoalProgressBar progress={0.46} />);
    const bar = screen.getByRole("progressbar");
    expect(bar).toHaveAttribute("aria-valuenow", "46");
  });
});

describe("AnnualGoalCard", () => {
  it("shows set-goal CTA when unset", () => {
    render(
      <AnnualGoalCard
        year={2026}
        goalAmount={null}
        ytdNetPnl={0}
        currency="USD"
        onSave={vi.fn<(...args: any[]) => any>(async () => {})}
      />,
    );
    expect(screen.getByText(/Set a 2026 net P&L target/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /set annual goal/i })).toBeInTheDocument();
  });

  it("renders hero progress when goal is set", () => {
    render(
      <AnnualGoalCard
        year={2026}
        goalAmount={100_000}
        ytdNetPnl={46_000}
        currency="USD"
        variant="hero"
        onSave={vi.fn<(...args: any[]) => any>(async () => {})}
        onClear={vi.fn<(...args: any[]) => any>(async () => {})}
      />,
    );
    expect(screen.getByText(/Annual P&L Goal/i)).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
    expect(screen.getByText(/vs linear pace/i)).toBeInTheDocument();
  });

  it("renders compact of-goal copy", () => {
    render(
      <AnnualGoalCard
        year={2026}
        goalAmount={100_000}
        ytdNetPnl={46_000}
        currency="USD"
        variant="compact"
        onSave={vi.fn<(...args: any[]) => any>(async () => {})}
      />,
    );
    expect(screen.getByText(/of/i)).toBeInTheDocument();
  });
  it("edits in display currency, restores canceled drafts and saves source units", async () => {
    const user = userEvent.setup();
    const save = vi.fn<(amount: number) => Promise<void>>(async () => {});
    render(
      <AnnualGoalCard
        year={2026}
        goalAmount={15000}
        ytdNetPnl={3000}
        currency="USD"
        fxRate={1 / 150}
        onSave={save}
      />,
    );
    await user.click(screen.getByRole("button", { name: /edit annual goal/i }));
    const input = screen.getByRole("textbox", { name: /Annual P&L goal amount/i });
    expect(input).toHaveValue("100");
    await user.clear(input);
    await user.type(input, "200");
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    expect(save).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: /edit annual goal/i }));
    expect(screen.getByRole("textbox", { name: /Annual P&L goal amount/i })).toHaveValue("100");
    await user.clear(screen.getByRole("textbox", { name: /Annual P&L goal amount/i }));
    await user.type(screen.getByRole("textbox", { name: /Annual P&L goal amount/i }), "200");
    await user.click(screen.getByRole("button", { name: /save goal/i }));
    expect(save.mock.calls[0][0]).toBeCloseTo(30000);
  });
  it("hides progress and retains a repair action when FX or legacy currency is unavailable", () => {
    render(
      <AnnualGoalCard
        year={2026}
        goalAmount={15000}
        ytdNetPnl={3000}
        currency="USD"
        unavailable="Legacy goal amount 15000 has no currency"
        onSave={vi.fn<(amount: number) => Promise<void>>(async () => {})}
      />,
    );
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(screen.getByText(/has no currency/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /edit annual goal/i })).toBeInTheDocument();
  });
});
