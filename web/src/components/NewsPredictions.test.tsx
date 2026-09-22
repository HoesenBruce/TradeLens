import { I18nProvider } from "@lingui/react";
import { i18n } from "@/i18n";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vite-plus/test";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { News } from "@/lib/api/news";
import { NewsPredictions, type PredictionActions } from "./NewsPredictions";

it("saves horizons, resets cancelled drafts, and preserves AI records", async () => {
  const user = userEvent.setup();
  const onSavePrediction = vi
    .fn<PredictionActions["onSavePrediction"]>()
    .mockResolvedValue(undefined);
  const news = {
    id: "n",
    title: "Catalyst",
    assets: [{ id: "a", symbol: "285A", asset_type: "stock" }],
    predictions: [
      {
        id: "p",
        news_asset_id: "a",
        source: "ai",
        direction: "bullish",
        confidence: 80,
        reasoning: "AI judgment",
        horizons: [1, 5],
      },
    ],
  } as News;
  render(
    <I18nProvider i18n={i18n}>
      <TooltipProvider>
        <NewsPredictions
          news={news}
          onSavePrediction={onSavePrediction}
          onDeletePrediction={vi.fn<PredictionActions["onDeletePrediction"]>()}
        />
      </TooltipProvider>
    </I18nProvider>,
  );
  expect(screen.queryByRole("button", { name: "Edit prediction" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Add prediction" }));
  await user.type(screen.getByRole("textbox", { name: "Reasoning" }), "Abandoned");
  await user.click(screen.getByRole("button", { name: "Cancel" }));
  await user.click(screen.getByRole("button", { name: "Add prediction" }));
  expect(screen.getByRole("textbox", { name: "Reasoning" })).toHaveValue("");
  await user.click(screen.getByRole("checkbox", { name: "1D" }));
  await user.click(screen.getByRole("button", { name: "Save prediction" }));
  expect(screen.getByRole("alert")).toHaveTextContent("Select at least one");
  await user.click(screen.getByRole("checkbox", { name: "3D" }));
  await user.click(screen.getByRole("checkbox", { name: "20D" }));
  await user.selectOptions(screen.getByRole("combobox", { name: "Direction" }), "bearish");
  await user.type(screen.getByRole("spinbutton", { name: "Confidence (%)" }), "100");
  await user.click(screen.getByRole("button", { name: "Save prediction" }));
  await waitFor(() =>
    expect(onSavePrediction).toHaveBeenCalledWith(
      "n",
      expect.objectContaining({ direction: "bearish", confidence: 100, horizons: [3, 20] }),
      undefined,
    ),
  );
});
