import { useUI } from "@/lib/ui";
import { useQueryClient } from "@tanstack/react-query";
import { act, render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { useState } from "react";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { useAuth } from "@/lib/auth";
import { useOcrSettings, useSaveOcrSettings } from "@/lib/hooks/useOcrSettings";
import { useLlmModelCache, LLM_MODEL_CACHE_STORAGE_KEY } from "@/lib/llmModelCache";
import { SessionQueryProvider } from "./SessionQueryProvider";

let client: ReturnType<typeof useQueryClient>;
function Probe() {
  client = useQueryClient();
  const settings = useOcrSettings();
  const save = useSaveOcrSettings();
  const [draft, setDraft] = useState("");
  return (
    <>
      <output>{settings.data?.custom_prompt ?? "empty"}</output>
      <input aria-label="draft" value={draft} onChange={(e) => setDraft(e.target.value)} />
      <button
        onClick={() =>
          save.mutate({
            enabled: true,
            base_url: "https://synthetic.invalid/v1",
            model: "synthetic",
            custom_prompt: "synthetic-save",
          })
        }
      >
        Save
      </button>
    </>
  );
}
function signIn(identity: string) {
  act(() => useAuth.getState().signIn(identity, "synthetic-refresh"));
}
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  useAuth.getState().signOut();
});

it("discards queries, mounted dirty state and model discovery across all identity transitions", async () => {
  vi.spyOn(globalThis, "fetch").mockImplementation(async (url, opts) => {
    const identity = new Headers(opts?.headers).get("Authorization")!.replace("Bearer ", "");
    if ((url instanceof Request ? url.url : url.toString()).endsWith("/me"))
      return Response.json({ id: identity, is_admin: identity.startsWith("owner") });
    return Response.json({ custom_prompt: identity });
  });
  signIn("owner-a");
  render(
    <SessionQueryProvider>
      <Probe />
    </SessionQueryProvider>,
  );
  await screen.findByText("owner-a");
  const oldClient = client!;
  act(() =>
    useUI.getState().openNoteEdit({
      id: "synthetic",
      type: "note",
      occurredAt: "2026-01-06",
      title: "synthetic-private",
      body: "synthetic-private",
      symbols: [],
    }),
  );
  act(() => {
    useLlmModelCache.getState().setModels("https://synthetic.invalid", ["model"]);
  });
  const input = screen.getByRole("textbox") as HTMLInputElement;
  fireEvent.change(input, { target: { value: "synthetic-dirty" } });
  expect(input).toHaveValue("synthetic-dirty");
  for (const identity of ["member", "owner-b", "owner-a"]) {
    signIn(identity);
    expect(screen.queryByText(identity === "member" ? "owner-a" : "member")).toBeNull();
    expect(screen.getByRole("textbox")).toHaveValue("");
    await screen.findByText(identity === "member" ? "empty" : identity);
  }
  expect(oldClient.getQueryCache().getAll()).toHaveLength(0);
  expect(useLlmModelCache.getState().modelsByBaseUrl).toEqual({});
  expect(useUI.getState().noteDraft).toBeNull();
  expect(useUI.getState().editTradeDetail).toBeNull();
  expect(useUI.getState().modal).toBeNull();
  expect(localStorage.getItem(LLM_MODEL_CACHE_STORAGE_KEY)).toBeNull();
  act(() => useAuth.getState().signOut());
  expect(screen.getByText("empty")).toBeVisible();
});

it("rejects late owner reads and mutation writes after switching to a member", async () => {
  let releaseRead!: () => void;
  let releaseSave!: () => void;
  let delayRead = true;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (url, opts) => {
    if ((url instanceof Request ? url.url : url.toString()).endsWith("/me"))
      return Response.json({ is_admin: true });
    if (opts?.method === "PUT") {
      await new Promise<void>((r) => {
        releaseSave = r;
      });
    } else if (delayRead) {
      await new Promise<void>((r) => {
        releaseRead = r;
      });
    }
    return Response.json({ custom_prompt: "synthetic-owner" });
  });
  signIn("owner");
  render(
    <SessionQueryProvider>
      <Probe />
    </SessionQueryProvider>,
  );
  await waitFor(() => expect(releaseRead).toBeTypeOf("function"));
  screen.getByRole("button", { name: "Save" }).click();
  await waitFor(() => expect(releaseSave).toBeTypeOf("function"));
  // Member /me denies owner privileges even when the old owner response arrives late.
  delayRead = false;
  vi.spyOn(globalThis, "fetch").mockResolvedValue(Response.json({ is_admin: false }));
  signIn("member");
  await act(async () => {
    releaseRead();
    releaseSave();
  });
  expect(screen.queryByText("synthetic-owner")).toBeNull();
  expect(client!.getQueryData(["settings", "ocr", useAuth.getState().session])).toBeUndefined();
});

it("drops private values when authentication expires during a background refetch", async () => {
  const { setUnauthorizedHandler } = await import("@/lib/api/client");
  setUnauthorizedHandler(() => useAuth.getState().signOut());
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(async (url) =>
      Response.json(
        (url instanceof Request ? url.url : url.toString()).endsWith("/me")
          ? { is_admin: true }
          : { custom_prompt: "synthetic-owner" },
      ),
    );
  signIn("owner");
  render(
    <SessionQueryProvider>
      <Probe />
    </SessionQueryProvider>,
  );
  await screen.findByText("synthetic-owner");
  fetch.mockResolvedValue(new Response("{}", { status: 401 }));
  await act(async () => {
    await client!.refetchQueries({ queryKey: ["settings", "ocr"] });
  });
  await waitFor(() => expect(useAuth.getState().authed).toBe(false));
  expect(screen.queryByText("synthetic-owner")).toBeNull();
  setUnauthorizedHandler(null);
});

it("does not expose cached settings after a background 403", async () => {
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(async (url) =>
      Response.json(
        (url instanceof Request ? url.url : url.toString()).endsWith("/me")
          ? { is_admin: true }
          : { custom_prompt: "synthetic-owner" },
      ),
    );
  signIn("owner");
  render(
    <SessionQueryProvider>
      <Probe />
    </SessionQueryProvider>,
  );
  await screen.findByText("synthetic-owner");
  fetch.mockResolvedValue(new Response("{}", { status: 403 }));
  await act(async () => {
    await client!.refetchQueries({ queryKey: ["settings", "ocr"] });
  });
  await waitFor(() => expect(screen.queryByText("synthetic-owner")).toBeNull());
});
