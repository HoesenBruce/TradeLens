import { describe, expect, it } from "vite-plus/test";
import { isApiPath, isNetworkOnlyRequest, isRetiredApiEntry } from "./workerCachePolicy";

describe("worker API policy", () => {
  it.each([
    "/api/v1/me",
    "/api",
    "/gateway/api/v1/admin/users",
    "/tenant/gateway/api/v1/admin/backup",
    "/gateway/api/v1/export.json",
    "/gateway/settings/ocr",
    "/gateway/trades/fixture/coach/reviews",
    "/gateway/news/fixture/analyze",
    "/gateway/news/export",
  ])("never caches API path %s", (path) => {
    expect(isApiPath(path)).toBe(true);
    expect(isNetworkOnlyRequest(new Request(`https://synthetic.invalid${path}`))).toBe(true);
  });

  it.each(["/favicon.svg", "/assets/app.js", "/api-logo.svg", "/apiary", "/gateway/apiculture"])(
    "preserves public asset path %s",
    (path) => expect(isApiPath(path)).toBe(false),
  );

  it("blocks authenticated custom/cross-origin bases independently of worker scope", () => {
    const request = new Request("https://synthetic.invalid/backend/tenant/me", {
      headers: { Authorization: "Bearer TEST_USER_A" },
    });
    expect(isNetworkOnlyRequest(request)).toBe(true);
    expect(isRetiredApiEntry(request)).toBe(true);
  });

  it("retires both unscoped API entries and old bearer-scoped entries", () => {
    for (const path of ["/gateway/api/v1/me", "/backend/me?__tm_auth_scope=synthetic"]) {
      expect(isRetiredApiEntry(new Request(`https://synthetic.invalid${path}`))).toBe(true);
    }
    expect(isRetiredApiEntry(new Request("https://synthetic.invalid/favicon.svg"))).toBe(false);
  });
});
