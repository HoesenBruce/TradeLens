import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { News } from "./api/news";
import { isoToWallClock } from "./displayPrefs";

export interface NewsFilters {
  status: "all" | "pending" | "validated";
  direction: "all" | "bullish" | "bearish" | "neutral";
  symbol: string;
  from: string;
  to: string;
}
export const defaultNewsFilters: NewsFilters = {
  status: "all",
  direction: "all",
  symbol: "",
  from: "",
  to: "",
};

// ponytail: filters scan the existing full News response; add API pagination when volume warrants it.
export function filterNews(news: News[], filters: NewsFilters): News[] {
  const symbol = filters.symbol.trim().toUpperCase();
  return news.filter((n) => {
    // Outcomes are supplied by the later validation feature; current records are pending.
    if (filters.status === "validated") return false;
    const date = isoToWallClock(n.published_at).slice(0, 10);
    if ((filters.from && date < filters.from) || (filters.to && date > filters.to)) return false;
    if (symbol && !n.assets.some((a) => a.symbol.toUpperCase().includes(symbol))) return false;
    return (
      filters.direction === "all" ||
      (n.predictions ?? []).some(
        (p) =>
          p.direction === filters.direction &&
          (!symbol ||
            n.assets.some(
              (a) => a.id === p.news_asset_id && a.symbol.toUpperCase().includes(symbol),
            )),
      )
    );
  });
}

export const useNewsFilters = create<{
  filters: NewsFilters;
  setFilters: (patch: Partial<NewsFilters>) => void;
  reset: () => void;
}>()(
  persist(
    (set) => ({
      filters: defaultNewsFilters,
      setFilters: (patch) => set((s) => ({ filters: { ...s.filters, ...patch } })),
      reset: () => set({ filters: defaultNewsFilters }),
    }),
    { name: "tm-news-filters", partialize: (s) => ({ filters: s.filters }) },
  ),
);
