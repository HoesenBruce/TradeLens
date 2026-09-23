import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { I18nProvider } from "@lingui/react";
import { i18n, loadLocale } from "@/i18n";
import { TooltipProvider } from "@/components/ui/tooltip";
import { newsAnalysisApi, type News } from "@/lib/api/news";
import { NewsAnalysisReview } from "./NewsAnalysisReview";

afterEach(async () => {
  vi.restoreAllMocks();
  await loadLocale("en");
});

it("keeps API enum values while reviewing and saving Japanese suggestions", async () => {
  await loadLocale("ja");
  const user = userEvent.setup();
  vi.spyOn(newsAnalysisApi, "analyze").mockResolvedValue({
    summary: "QA summary",
    category: "QA category",
    assets: [
      {
        asset_type: "etf",
        symbol: "SPY",
        market: "US",
        exchange: "NYSE",
        display_name: "QA",
        direction: "neutral",
        confidence: 50,
        reasoning: "QA",
        catalysts: "",
        risks: "",
        horizons: [1],
      },
    ],
  });
  const accept = vi.spyOn(newsAnalysisApi, "accept").mockResolvedValue({} as News);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <I18nProvider i18n={i18n}>
        <TooltipProvider>
          <NewsAnalysisReview news={{ id: "n", summary: "", category: "" } as News} />
        </TooltipProvider>
      </I18nProvider>
    </QueryClientProvider>,
  );
  await user.click(screen.getByRole("button", { name: "AI で分析" }));
  expect(await screen.findByRole("combobox", { name: "種類" })).toHaveValue("etf");
  expect(screen.getByRole("combobox", { name: "方向" })).toHaveValue("neutral");
  await user.selectOptions(screen.getByRole("combobox", { name: "種類" }), "stock");
  await user.selectOptions(screen.getByRole("combobox", { name: "方向" }), "bullish");
  await user.click(screen.getByRole("checkbox", { name: "資産 1 を採用 · AI" }));
  await user.click(screen.getByRole("button", { name: "選択した提案を保存" }));
  await waitFor(() =>
    expect(accept).toHaveBeenCalledWith(
      "n",
      expect.objectContaining({
        assets: [
          expect.objectContaining({ asset_type: "stock", direction: "bullish", source: "ai" }),
        ],
      }),
    ),
  );
});
