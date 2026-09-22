import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@lingui/react";
import { Blob as NodeBlob } from "node:buffer";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { i18n } from "@/i18n";
import { downloadNews } from "@/lib/api/news";
import { NewsExportActions } from "./NewsExportActions";

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it("downloads unchanged Markdown with server filename and deduplicated IDs, rejecting empty/errors", async () => {
  const fetch = vi.spyOn(globalThis, "fetch");
  vi.stubGlobal("Blob", NodeBlob);
  const create = vi.fn<(blob: Blob) => string>().mockReturnValue("blob:export");
  vi.stubGlobal(
    "URL",
    class extends URL {
      static createObjectURL = create;
      static revokeObjectURL = vi.fn<() => void>();
    },
  );
  const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  const markdown = "---\ntype: news-thesis-batch\ncount: 2\n---\nServer Markdown";
  fetch.mockResolvedValue(
    new Response(markdown, {
      headers: { "Content-Disposition": 'attachment; filename="server.md"' },
    }),
  );
  await downloadNews(["a", "b", "a"]);
  expect(fetch.mock.calls[0][0]).toBe("/api/v1/news/export?id=a&id=b");
  expect(await (create.mock.calls[0][0] as Blob).text()).toBe(markdown);
  expect((click.mock.instances[0] as HTMLAnchorElement).download).toBe("server.md");
  await expect(downloadNews([])).rejects.toMatchObject({ code: "empty_export" });
  expect(fetch).toHaveBeenCalledTimes(1);
  for (const body of ["", "---\ntype: news-thesis-batch\ncount: 0\n---\nNo matches"]) {
    fetch.mockResolvedValue(new Response(body));
    await expect(downloadNews(["a"])).rejects.toMatchObject({ code: "empty_export" });
  }
  for (const status of [400, 401, 403, 404, 500]) {
    fetch.mockResolvedValue(
      new Response('{"error":{"code":"test","message":"failure"}}', { status }),
    );
    await expect(downloadNews("a")).rejects.toMatchObject({ status });
  }
  expect(fetch.mock.calls.at(-1)?.[0]).toBe("/api/v1/news/a/export");
  fetch.mockResolvedValue(new Response("Markdown"));
  click.mockImplementation(() => {
    throw new Error("Browser download failed");
  });
  await expect(downloadNews("a")).rejects.toThrow("Browser download failed");
  expect(document.querySelector("a[download]")).toBeNull();
});

it("blocks repeated clicks while pending, shows failures, and enables retry", async () => {
  vi.stubGlobal("Blob", NodeBlob);
  let finish!: (res: Response) => void;
  const fetch = vi.spyOn(globalThis, "fetch").mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  render(
    <I18nProvider i18n={i18n}>
      <NewsExportActions ids={["a", "b"]} selected={["a"]} />
    </I18nProvider>,
  );
  const selected = screen.getByRole("button", { name: "Export selected (1)" });
  fireEvent.click(selected);
  fireEvent.click(selected);
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(selected).toBeDisabled();
  expect(screen.getByRole("button", { name: "Export filtered" })).toBeDisabled();
  expect(screen.getByText("Exporting…")).toBeVisible();
  finish(new Response("{}", { status: 404 }));
  await waitFor(() => expect(selected).toBeEnabled());
  expect(screen.getByText(/A thesis is no longer available/)).toBeVisible();
  fetch.mockResolvedValue(new Response(""));
  await userEvent.click(selected);
  expect(await screen.findByText("No theses to export.")).toBeVisible();
});
