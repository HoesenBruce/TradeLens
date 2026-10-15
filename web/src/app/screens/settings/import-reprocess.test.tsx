import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@/test/render";
import { Toaster } from "@/components/Toaster";
import { importsApi } from "@/lib/api/imports";
import type { Account, ImportBatch, ReprocessResult } from "@/lib/api/types";
import { ImportHistorySection } from "./import-history";

vi.mock("@/lib/api/imports", () => ({ importsApi: { list: vi.fn(), reprocess: vi.fn() } }));
const batch: ImportBatch = {
  id: "b1",
  user_id: "u1",
  account_id: "a1",
  source: "csv",
  filename: "fills.csv",
  column_mapping: null,
  row_count: 2,
  retained_rows: 2,
  reprocessable: true,
  status: "committed",
  created_at: "2026-10-01T00:00:00Z",
};
const result: ReprocessResult = {
  retained_rows: 2,
  trades_changed: 1,
  executions: 2,
  changed: 1,
  trades: 1,
  fingerprint: "preview-1",
  applied: false,
};

function mount() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <Toaster>
        <ImportHistorySection accounts={[{ id: "a1", name: "Main" } as Account]} />
      </Toaster>
    </QueryClientProvider>,
  );
  return client;
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(importsApi.list).mockResolvedValue([batch]);
});

async function open() {
  fireEvent.click(await screen.findByRole("button", { name: /^Reprocess$/ }));
  return screen.getByRole("dialog", { name: "Reprocess with current rules" });
}
describe("Import reprocessing", () => {
  it("requires preview, applies its fingerprint, and refreshes calculated data only after apply", async () => {
    vi.mocked(importsApi.reprocess)
      .mockResolvedValueOnce(result)
      .mockResolvedValueOnce({ ...result, applied: true });
    const client = mount();
    const invalidation = vi.spyOn(client, "invalidateQueries");
    const dialog = await open();
    expect(
      within(dialog).queryByRole("button", { name: "Apply reprocessing" }),
    ).not.toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Preview changes" }));
    expect(
      await within(dialog).findByText(
        "2 source rows · 2 executions · 1 changed · 1 trades changed",
      ),
    ).toBeInTheDocument();
    expect(importsApi.reprocess).toHaveBeenLastCalledWith("b1", undefined);
    expect(invalidation).not.toHaveBeenCalled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Apply reprocessing" }));
    await waitFor(() => expect(importsApi.reprocess).toHaveBeenLastCalledWith("b1", "preview-1"));
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Reprocess with current rules" }),
      ).not.toBeInTheDocument(),
    );
    expect(invalidation).toHaveBeenCalledWith({ queryKey: ["trade"] });
    expect(invalidation).toHaveBeenCalledWith({ queryKey: ["analytics"] });
  });
  it("cancel and re-entry discard the abandoned preview", async () => {
    vi.mocked(importsApi.reprocess).mockResolvedValue(result);
    mount();
    let dialog = await open();
    fireEvent.click(within(dialog).getByRole("button", { name: "Preview changes" }));
    await within(dialog).findByRole("button", { name: "Apply reprocessing" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Reprocess with current rules" }),
      ).not.toBeInTheDocument(),
    );
    dialog = await open();
    expect(within(dialog).getByRole("button", { name: "Preview changes" })).toBeInTheDocument();
    expect(importsApi.reprocess).toHaveBeenCalledTimes(1);
  });
  it("shows conflicts and requires a fresh preview before retry", async () => {
    vi.mocked(importsApi.reprocess).mockRejectedValueOnce(new Error("Existing edits were kept"));
    mount();
    const dialog = await open();
    fireEvent.click(within(dialog).getByRole("button", { name: "Preview changes" }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("Existing edits were kept");
    expect(within(dialog).getByRole("button", { name: "Preview changes" })).toBeInTheDocument();
  });
  it("explains unavailable legacy/revised sources and hides replay for rolled-back imports", async () => {
    vi.mocked(importsApi.list).mockResolvedValue([
      { ...batch, reprocessable: false, retained_rows: 0 },
      { ...batch, id: "b2", filename: "revised.csv", reprocessable: false },
      { ...batch, id: "b3", filename: "reversed.csv", status: "reversed", reprocessable: false },
    ]);
    mount();
    await screen.findByText(/Older, duplicate-only or unsupported imports/);
    expect(screen.getByText("Newer source data exists; use its import batch.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Reprocess$/ })).not.toBeInTheDocument();
  });
});
