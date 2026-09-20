import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import { useUpdateNews } from "./useNews";

const api = vi.hoisted(() => ({
  update: vi.fn<(id: string, body: unknown) => Promise<unknown>>(),
  updateAsset: vi.fn<(newsId: string, id: string, body: unknown) => Promise<unknown>>(),
  createAsset: vi.fn<(newsId: string, body: unknown) => Promise<unknown>>(),
  deleteAsset: vi.fn<(newsId: string, id: string) => Promise<void>>(),
  get: vi.fn<(id: string) => Promise<unknown>>(),
}));

vi.mock("@/lib/api/news", () => ({ newsApi: api }));

describe("useUpdateNews", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    for (const fn of [api.update, api.updateAsset, api.createAsset, api.deleteAsset, api.get]) {
      fn.mockResolvedValue(undefined);
    }
  });

  it("updates kept assets, creates new ones, and removes deleted ones", async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useUpdateNews(), { wrapper });

    await act(() =>
      result.current.mutateAsync({
        id: "news-1",
        body: { title: "Updated", source: "manual", published_at: "2026-09-20T09:30:00Z" },
        previousAssetIds: ["asset-1", "asset-2"],
        assets: [
          { id: "asset-1", asset_type: "stock", symbol: "285A", source: "user" },
          { asset_type: "etf", symbol: "1306", source: "user" },
        ],
      }),
    );

    expect(api.updateAsset).toHaveBeenCalledWith("news-1", "asset-1", {
      asset_type: "stock",
      symbol: "285A",
      source: "user",
    });
    expect(api.createAsset).toHaveBeenCalledWith("news-1", {
      asset_type: "etf",
      symbol: "1306",
      source: "user",
    });
    expect(api.deleteAsset).toHaveBeenCalledWith("news-1", "asset-2");
  });
});
