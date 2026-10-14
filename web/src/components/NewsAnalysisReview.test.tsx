import { render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { I18nProvider } from "@lingui/react";
import { i18n, loadLocale } from "@/i18n";
import { newsAnalysisApi, type News } from "@/lib/api/news";
import { NewsAnalysisReview } from "./NewsAnalysisReview";

afterEach(() => vi.restoreAllMocks());

it("shows fixed unavailability without calling legacy analysis", async () => {
  await loadLocale("en");
  vi.spyOn(globalThis, "fetch").mockImplementation(
    async () =>
      new Response(JSON.stringify({ available: false, news_analysis_available: false }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
  );
  const analyze = vi.spyOn(newsAnalysisApi, "analyze");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <I18nProvider i18n={i18n}>
        <NewsAnalysisReview news={{ id: "n", summary: "", category: "" } as News} />
      </I18nProvider>
    </QueryClientProvider>,
  );
  expect(screen.getByRole("status")).toHaveTextContent("Manual news review remains available");
  expect(screen.queryByRole("button", { name: "Analyze with AI" })).not.toBeInTheDocument();
  expect(analyze).not.toHaveBeenCalled();
  client.clear();
});
