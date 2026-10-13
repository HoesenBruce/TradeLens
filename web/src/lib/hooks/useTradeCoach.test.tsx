import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { useAuth } from "@/lib/auth";
import { tradeCoachReviewsKey, useTradeCoach } from "./useTradeCoach";

afterEach(() => vi.restoreAllMocks());

it("withholds old query bytes across logout and user change", async () => {
  const fetch = vi.spyOn(globalThis, "fetch").mockImplementation(
    async () =>
      new Response(
        JSON.stringify({
          available: false,
          news_analysis_available: false,
          reason: "unavailable",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(tradeCoachReviewsKey("trade"), {
    reviews: [
      {
        notes: [{ headline: "PRIVATE-LEGACY" }],
        model: "PRIVATE-LEGACY",
        next_action: "PRIVATE-LEGACY",
      },
    ],
  });
  client.setQueryData(["settings", "coach"], { enabled: true });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(() => useTradeCoach("trade"), { wrapper });
  await waitFor(() => expect(result.current.settingsPending).toBe(false));
  expect(result.current.data).toBeUndefined();
  expect(result.current.history).toEqual([]);
  expect(result.current.streamingNotes).toEqual([]);
  expect(result.current.coachConfigured).toBe(false);
  await expect(result.current.generate()).rejects.toThrow("unavailable");
  act(() => useAuth.getState().signOut());
  act(() => useAuth.getState().signIn("synthetic-user-token", ""));
  await waitFor(() => expect(result.current.settingsPending).toBe(false));
  expect(result.current.data).toBeUndefined();
  expect(
    fetch.mock.calls.every(
      ([url]) => typeof url === "string" && url.endsWith("/capabilities/coach"),
    ),
  ).toBe(true);
  client.clear();
});
