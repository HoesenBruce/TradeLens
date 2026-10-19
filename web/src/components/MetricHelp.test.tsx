import { I18nProvider } from "@lingui/react";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { i18n, loadLocale } from "@/i18n";
import { MetricHelp, MetricLabel } from "./MetricHelp";
import { metricDefinitions } from "@/lib/metrics/definitions";

afterEach(cleanup);
it("opens by keyboard, reads the definition and restores focus on Escape", async () => {
  await loadLocale("en");
  const user = userEvent.setup();
  const activateRow = vi.fn<() => void>();
  render(
    <I18nProvider i18n={i18n}>
      <div
        onKeyDown={(event) => {
          if (event.key === "Enter") activateRow();
        }}
      >
        <MetricHelp metric="holding" />
      </div>
    </I18nProvider>,
  );
  const trigger = screen.getByRole("button", { name: "Explain Holding duration" });
  await user.tab();
  await waitFor(() => expect(trigger).toHaveFocus());
  await user.keyboard("{Enter}");
  expect(await screen.findByRole("dialog")).toHaveAccessibleName("Holding duration");
  expect(screen.getByText("Computation")).toBeVisible();
  trigger.focus();
  await user.keyboard("{Escape}");
  await waitFor(() => expect(trigger).toHaveFocus());
  await user.click(trigger);
  await user.click(screen.getByRole("button", { name: "Close explanation" }));
  await waitFor(() => expect(trigger).toHaveFocus());
  expect(activateRow).not.toHaveBeenCalled();
});

it.each([
  "accountPerformance",
  "accountRealizedPnl",
  "pnl",
  "winRate",
  "fees",
  "drawdown",
] as const)(
  "uses a neutral computation heading and preserves the %s explanation",
  async (metric) => {
    await loadLocale("en");
    const user = userEvent.setup();
    const entry = metricDefinitions[metric];
    render(
      <I18nProvider i18n={i18n}>
        <MetricHelp metric={metric} />
      </I18nProvider>,
    );
    await user.click(screen.getByRole("button", { name: `Explain ${i18n._(entry.title)}` }));
    expect(screen.getByText("Computation", { exact: true })).toBeVisible();
    expect(screen.queryByText("Computation / denominator")).not.toBeInTheDocument();
    expect(screen.getByText(i18n._(entry.formula), { exact: true })).toBeVisible();
    expect(screen.getByText(i18n._(entry.caveat), { exact: true })).toBeVisible();
  },
);

it("groups the visible label with its own help and omits help for ordinary labels", async () => {
  await loadLocale("en");
  const user = userEvent.setup();
  render(
    <I18nProvider i18n={i18n}>
      <MetricLabel metric="holding">Visible holding label</MetricLabel>
      <MetricLabel>Ordinary label</MetricLabel>
    </I18nProvider>,
  );
  const trigger = screen.getByRole("button", { name: "Explain Holding duration" });
  expect(trigger.closest("[data-metric-label]")).toHaveTextContent("Visible holding label");
  expect(screen.getAllByRole("button")).toHaveLength(1);
  await user.click(trigger);
  expect(await screen.findByRole("dialog")).toHaveAccessibleName("Holding duration");
  await user.click(screen.getByRole("button", { name: "Close explanation" }));
  await waitFor(() => expect(trigger).toHaveFocus());
});
