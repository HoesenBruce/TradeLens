import { apiFetch } from "./client";

export type NewsAssetType = "stock" | "etf" | "index";

export interface NewsAssetBody {
  asset_type: NewsAssetType;
  symbol: string;
  market?: string;
  exchange?: string;
  display_name?: string;
  relation?: string;
  source?: "user" | "ai";
}

export interface NewsAsset extends Required<NewsAssetBody> {
  id: string;
  news_id: string;
  created_at: string;
  updated_at: string;
}

export interface NewsBody {
  title: string;
  source: string;
  url?: string;
  published_at: string;
  original_text?: string;
  notes?: string;
  summary?: string;
  category?: string;
  tags?: string[];
  assets?: NewsAssetBody[];
}

export interface News extends Required<Omit<NewsBody, "assets">> {
  id: string;
  user_id: string;
  assets: NewsAsset[];
  created_at: string;
  updated_at: string;
}

export const newsApi = {
  list: () => apiFetch<News[]>("/news"),
  get: (id: string) => apiFetch<News>(`/news/${id}`),
  create: (body: NewsBody) =>
    apiFetch<News>("/news", { method: "POST", body: JSON.stringify(body) }),
  update: (id: string, body: NewsBody) =>
    apiFetch<News>(`/news/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  delete: (id: string) => apiFetch<void>(`/news/${id}`, { method: "DELETE" }),
  createAsset: (newsId: string, body: NewsAssetBody) =>
    apiFetch<NewsAsset>(`/news/${newsId}/assets`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateAsset: (newsId: string, id: string, body: NewsAssetBody) =>
    apiFetch<NewsAsset>(`/news/${newsId}/assets/${id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  deleteAsset: (newsId: string, id: string) =>
    apiFetch<void>(`/news/${newsId}/assets/${id}`, { method: "DELETE" }),
};
