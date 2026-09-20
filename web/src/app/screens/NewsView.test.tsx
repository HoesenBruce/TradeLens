import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vite-plus/test";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { News } from "@/lib/api/news";
import { NewsView, type NewsFormValue } from "./NewsView";

const item: News = {
  id: "news-1",
  user_id: "user-1",
  title: "Japan ETF launch",
  source: "Company filing",
  url: "https://example.com/news",
  published_at: "2026-09-20T09:30:00Z",
  original_text: "Original filing text",
  notes: "Watch fund flows",
  summary: "A new ETF launched.",
  category: "Launch",
  tags: ["Japan", "ETF"],
  assets: [
    {
      id: "asset-1",
      news_id: "news-1",
      asset_type: "stock",
      symbol: "285A",
      market: "JP",
      exchange: "TSE",
      display_name: "Example stock",
      relation: "peer",
      source: "user",
      created_at: "2026-09-20T10:00:00Z",
      updated_at: "2026-09-20T10:00:00Z",
    },
    {
      id: "asset-2",
      news_id: "news-1",
      asset_type: "index",
      symbol: "N225",
      market: "JP",
      exchange: "",
      display_name: "Nikkei 225",
      relation: "benchmark",
      source: "user",
      created_at: "2026-09-20T10:00:00Z",
      updated_at: "2026-09-20T10:00:00Z",
    },
  ],
  created_at: "2026-09-20T10:00:00Z",
  updated_at: "2026-09-20T10:00:00Z",
};

function renderView(props: Partial<ComponentProps<typeof NewsView>> = {}) {
  return render(
    <TooltipProvider>
      <NewsView
        news={[item]}
        loading={false}
        error={false}
        onRetry={vi.fn<() => void>()}
        onSave={vi
          .fn<(value: NewsFormValue, item?: News) => Promise<void>>()
          .mockResolvedValue(undefined)}
        onDelete={vi.fn<(item: News) => Promise<void>>().mockResolvedValue(undefined)}
        {...props}
      />
    </TooltipProvider>,
  );
}

describe("NewsView", () => {
  it("renders loading, error, and empty states", async () => {
    const onRetry = vi.fn<() => void>();
    const view = renderView({ news: [], loading: true });
    expect(document.querySelector("[data-slot=skeleton]")).toBeInTheDocument();

    view.rerender(
      <TooltipProvider>
        <NewsView
          news={[]}
          loading={false}
          error
          onRetry={onRetry}
          onSave={vi.fn<(value: NewsFormValue, item?: News) => Promise<void>>()}
          onDelete={vi.fn<(item: News) => Promise<void>>()}
        />
      </TooltipProvider>,
    );
    await userEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetry).toHaveBeenCalledOnce();

    view.rerender(
      <TooltipProvider>
        <NewsView
          news={[]}
          loading={false}
          error={false}
          onRetry={onRetry}
          onSave={vi.fn<(value: NewsFormValue, item?: News) => Promise<void>>()}
          onDelete={vi.fn<(item: News) => Promise<void>>()}
        />
      </TooltipProvider>,
    );
    expect(screen.getByText("No news theses yet")).toBeInTheDocument();
  });

  it("creates an entry with an alphanumeric affected asset", async () => {
    const user = userEvent.setup();
    const onSave = vi
      .fn<(value: NewsFormValue, item?: News) => Promise<void>>()
      .mockResolvedValue(undefined);
    renderView({ news: [], onSave });

    await user.click(screen.getAllByRole("button", { name: "New entry" })[0]);
    await user.type(screen.getByRole("textbox", { name: "Title" }), "AI supply agreement");
    await user.type(screen.getByRole("textbox", { name: "Source" }), "Company release");
    await user.click(screen.getByRole("button", { name: "Add asset" }));
    await user.type(screen.getByRole("textbox", { name: "Symbol" }), "285a");
    await user.click(screen.getByRole("button", { name: "Create entry" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledOnce());
    const [value, editing] = onSave.mock.calls[0];
    expect(editing).toBeUndefined();
    expect(value.assets).toMatchObject([{ asset_type: "stock", symbol: "285A", source: "user" }]);
  });

  it("resets abandoned edits on re-entry and saves asset add/remove changes", async () => {
    const user = userEvent.setup();
    const onSave = vi
      .fn<(value: NewsFormValue, item?: News) => Promise<void>>()
      .mockResolvedValue(undefined);
    renderView({ onSave });

    await user.click(screen.getByRole("button", { name: "Edit Japan ETF launch" }));
    const title = screen.getByRole("textbox", { name: "Title" });
    await user.clear(title);
    await user.type(title, "Abandoned title");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onSave).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Edit Japan ETF launch" }));
    expect(screen.getByRole("textbox", { name: "Title" })).toHaveValue("Japan ETF launch");
    await user.click(screen.getByRole("button", { name: "Remove asset 2" }));
    await user.click(screen.getByRole("button", { name: "Add asset" }));
    await user.type(screen.getAllByRole("textbox", { name: "Symbol" })[1], "1306");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledOnce());
    const [value, editing] = onSave.mock.calls[0];
    expect(editing?.id).toBe(item.id);
    expect(value.assets.map((asset) => [asset.id, asset.symbol])).toEqual([
      ["asset-1", "285A"],
      [undefined, "1306"],
    ]);
  });

  it("cancels and confirms deletion in the in-app dialog", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn<(item: News) => Promise<void>>().mockResolvedValue(undefined);
    renderView({ onDelete });

    await user.click(screen.getByRole("button", { name: "Delete Japan ETF launch" }));
    expect(screen.getByText("Delete news thesis?")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onDelete).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Delete Japan ETF launch" }));
    await user.click(screen.getByRole("button", { name: /^Delete$/ }));
    await waitFor(() => expect(onDelete).toHaveBeenCalledWith(item));
  });
});
