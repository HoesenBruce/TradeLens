import { expect, it } from "vite-plus/test";
import type { News } from "./api/news";
import { defaultNewsFilters, filterNews } from "./newsFilters";

it("combines symbol and direction on the same asset and filters dates/status", () => {
  const news = [
    {
      id: "n",
      published_at: "2026-09-20T12:00:00Z",
      assets: [
        { id: "a", symbol: "285A" },
        { id: "b", symbol: "N225" },
      ],
      predictions: [
        { news_asset_id: "a", direction: "bullish" },
        { news_asset_id: "b", direction: "bearish" },
      ],
    },
  ] as News[];
  expect(filterNews(news, defaultNewsFilters)).toHaveLength(1);
  expect(
    filterNews(news, { ...defaultNewsFilters, symbol: "285a", direction: "bullish" }),
  ).toHaveLength(1);
  expect(
    filterNews(news, { ...defaultNewsFilters, symbol: "285a", direction: "bearish" }),
  ).toHaveLength(0);
  expect(filterNews(news, { ...defaultNewsFilters, to: "2026-09-01" })).toHaveLength(0);
  expect(filterNews(news, { ...defaultNewsFilters, from: "2026-10-01" })).toHaveLength(0);
  expect(filterNews(news, { ...defaultNewsFilters, status: "validated" })).toHaveLength(0);
  expect(filterNews(news, { ...defaultNewsFilters, status: "pending" })).toHaveLength(1);
});
