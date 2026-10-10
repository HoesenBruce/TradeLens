import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vite-plus/test";
import { useAuth } from "@/lib/auth";
import { useMe } from "./useMe";

const session = vi.hoisted(() => ({ token: "owner" }));
vi.mock("@/lib/api/client", () => ({
  getToken: () => session.token,
  setTokens: vi.fn<(access: string, refresh: string) => void>(),
}));
vi.mock("@/lib/api/auth", () => ({
  authApi: {
    me: () => Promise.resolve({ id: session.token, is_admin: session.token === "owner" }),
  },
}));

it("does not reuse the owner's identity after a member signs in", async () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(() => useMe(), { wrapper });
  await waitFor(() => expect(result.current.data?.is_admin).toBe(true));
  session.token = "member";
  act(() => useAuth.getState().signIn("member", "refresh"));
  expect(result.current.data?.is_admin).not.toBe(true);
  await waitFor(() => expect(result.current.data?.id).toBe("member"));
  expect(result.current.data?.is_admin).toBe(false);
});
