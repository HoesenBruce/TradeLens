import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vite-plus/test";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { News } from "@/lib/api/news";
import { NewsView, type NewsFormValue } from "./NewsView";
import { I18nProvider } from "@lingui/react";
import { i18n } from "@/i18n";

function LocalizedTooltipProvider({ children }: { children: React.ReactNode }) {
  return (
    <I18nProvider i18n={i18n}>
      <TooltipProvider>{children}</TooltipProvider>
    </I18nProvider>
  );
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, params }: { children: React.ReactNode; params: { id: string } }) => (
    <a href={`/news/${params.id}`}>{children}</a>
  ),
}));

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
    <LocalizedTooltipProvider>
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
    </LocalizedTooltipProvider>,
  );
}

describe("NewsView", () => {
  it("keeps selection page-local and clears it on pagination, filtering, and re-entry", async () => {
    const user = userEvent.setup();
    const news = Array.from({ length: 12 }, (_, index) => ({
      ...item,
      id: `news-${index}`,
      title: `Thesis ${index}`,
    }));
    const view = renderView({ news });
    const select = () => screen.getByRole("checkbox", { name: "Select Thesis 0" });
    await user.click(select());
    expect(screen.getByRole("button", { name: "Export selected (1)" })).toBeEnabled();
    await user.click(select());
    expect(screen.getByRole("button", { name: "Export selected (0)" })).toBeDisabled();
    await user.click(select());
    await user.click(screen.getByRole("button", { name: "Next page" }));
    expect(screen.getByRole("button", { name: "Export selected (0)" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Previous page" }));
    expect(select()).not.toBeChecked();
    await user.click(select());
    await user.selectOptions(screen.getByRole("combobox", { name: "Rows per page" }), "20");
    expect(select()).not.toBeChecked();
    await user.click(select());
    await user.type(screen.getByRole("textbox", { name: "Symbol" }), "NO-MATCH");
    expect(screen.getByRole("button", { name: "Export filtered" })).toBeDisabled();
    expect(screen.getByText("No theses to export.")).toBeVisible();
    await user.click(screen.getAllByRole("button", { name: "Reset filters" })[0]);
    expect(select()).not.toBeChecked();
    await user.click(select());
    view.unmount();
    renderView({ news });
    expect(select()).not.toBeChecked();
  });

  it("renders loading, error, and empty states", async () => {
    const onRetry = vi.fn<() => void>();
    const view = renderView({ news: [], loading: true });
    expect(document.querySelector("[data-slot=skeleton]")).toBeInTheDocument();

    view.rerender(
      <LocalizedTooltipProvider>
        <NewsView
          news={[]}
          loading={false}
          error
          onRetry={onRetry}
          onSave={vi.fn<(value: NewsFormValue, item?: News) => Promise<void>>()}
          onDelete={vi.fn<(item: News) => Promise<void>>()}
        />
      </LocalizedTooltipProvider>,
    );
    await userEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetry).toHaveBeenCalledOnce();

    view.rerender(
      <LocalizedTooltipProvider>
        <NewsView
          news={[]}
          loading={false}
          error={false}
          onRetry={onRetry}
          onSave={vi.fn<(value: NewsFormValue, item?: News) => Promise<void>>()}
          onDelete={vi.fn<(item: News) => Promise<void>>()}
        />
      </LocalizedTooltipProvider>,
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
