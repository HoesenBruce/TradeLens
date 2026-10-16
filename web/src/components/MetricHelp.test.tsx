import { I18nProvider } from "@lingui/react";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { i18n, loadLocale } from "@/i18n";
import { MetricHelp } from "./MetricHelp";

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
  expect(screen.getByText("Computation / denominator")).toBeVisible();
  trigger.focus();
  await user.keyboard("{Escape}");
  await waitFor(() => expect(trigger).toHaveFocus());
  await user.click(trigger);
  await user.click(screen.getByRole("button", { name: "Close explanation" }));
  await waitFor(() => expect(trigger).toHaveFocus());
  expect(activateRow).not.toHaveBeenCalled();
});
