import { apiFetch } from "./client";

export const performanceFilterKeys = [
  "source",
  "symbol",
  "asset_type",
  "category",
  "horizon",
  "from",
  "to",
] as const;
export type PerformanceFilters = Partial<Record<(typeof performanceFilterKeys)[number], string>>;
export interface PerformanceCounts {
  total: number;
  pending: number;
  validated: number;
  unavailable: number;
  incomplete: number;
}
export interface PerformanceGroup extends PerformanceCounts {
  key: string;
  source: "ai" | "user";
  correct: number;
  sample_count: number;
  hit_rate: number | null;
}
export interface NewsPerformance {
  unit: "prediction_horizon";
  filters: Omit<PerformanceFilters, "horizon"> & { horizon: number };
  counts: PerformanceCounts;
  by_source: PerformanceGroup[];
  by_horizon: PerformanceGroup[];
  by_asset: PerformanceGroup[];
  by_category: PerformanceGroup[];
}
export function getNewsPerformance(filters: PerformanceFilters) {
  const query = new URLSearchParams();
  for (const key of performanceFilterKeys) if (filters[key]) query.set(key, filters[key]);
  return apiFetch<NewsPerformance>(`/news/performance?${query}`);
}
