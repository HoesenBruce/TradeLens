import { readFileSync } from "node:fs";
import { createServer } from "node:http";
import { resolve, extname } from "node:path";
import { expect, test } from "@playwright/test";

const api = process.env.E2E_API_URL ?? "http://localhost:8099/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5189";

test("real API availability and preserved manual surfaces", async ({ page, request }) => {
  const auth = await (
    await request.post(`${api}/auth/login`, {
      data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
    })
  ).json();
  const fixture = JSON.parse(readFileSync(process.env.E2E_COACH_FIXTURE!, "utf8"));
  const baselineNews = await (
    await request.get(`${api}/news/${fixture.news}`, {
      headers: { Authorization: `Bearer ${auth.access_token}` },
    })
  ).json();
  const editedTitle =
    baselineNews.title === "Issue 299 Manual News"
      ? "Issue 299 Manual News saved"
      : "Issue 299 Manual News";
  await page.addInitScript(
    ({ token, api }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
    },
    { token: auth.access_token, api },
  );
  await page.goto(`${web}/trades/${fixture.trade}`);
  await expect(
    page.getByText("AI Coach is temporarily unavailable. Rule-based review remains available.", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.getByText("PRIVATE-LEGACY-MARKER", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Ask AI coach", exact: true })).toHaveCount(0);
  await expect(
    page.getByText("Issue 299 manual review remains usable", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Coach", exact: true }).click();
  await expect(page.getByRole("button", { name: "Coach", exact: true })).toHaveAttribute(
    "aria-expanded",
    "false",
  );
  const coachPanel = page.locator('[data-slot="collapsible-content"]').filter({
    has: page.getByText(
      "AI Coach is temporarily unavailable. Rule-based review remains available.",
      { exact: true },
    ),
  });
  await expect(coachPanel).toHaveAttribute("aria-hidden", "true");
  await expect.poll(() => coachPanel.evaluate((el) => el.getBoundingClientRect().height)).toBe(0);
  await page.getByRole("button", { name: "Coach", exact: true }).click();
  await expect(
    page.getByText("AI Coach is temporarily unavailable. Rule-based review remains available.", {
      exact: true,
    }),
  ).toBeVisible();
  await page.goto(`${web}/news/${fixture.news}`);
  await expect(
    page.getByText(
      "AI news analysis is temporarily unavailable. Manual news review remains available.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Analyze with AI", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Edit news thesis", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("textbox", { name: "Title", exact: true }).fill("abandoned draft");
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await page.getByRole("button", { name: "Edit news thesis", exact: true }).click();
  await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toHaveValue(
    baselineNews.title,
  );
  await dialog.getByRole("textbox", { name: "Title", exact: true }).fill(editedTitle);
  await dialog.getByRole("button", { name: "Save changes", exact: true }).click();
  await page.reload();
  await expect(page.getByRole("heading", { name: editedTitle, exact: true })).toBeVisible();
  await page.goto(`${web}/settings#ai`);
  await expect(
    page.getByText("AI Coach is temporarily unavailable. Rule-based review remains available.", {
      exact: true,
    }),
  ).toBeVisible();
});

test("security worker automatically replaces old caching worker and purges legacy bytes", async ({
  page,
  context,
}) => {
  test.setTimeout(60000);
  let upgraded = false;
  const dist = resolve(process.env.E2E_WORKER_DIST ?? "dist");
  const worker = readFileSync(resolve(dist, "sw.js"));
  const legacy = `self.addEventListener('install',()=>self.skipWaiting());self.addEventListener('activate',e=>e.waitUntil(self.clients.claim()));self.addEventListener('fetch',e=>{if(e.request.url.includes('/coach/reviews'))e.respondWith(caches.match(e.request).then(r=>r||fetch(e.request)));});`;
  const server = createServer((req, res) => {
    const path = (req.url ?? "/").split("?")[0];
    res.setHeader("Cache-Control", "no-store");
    if (path === "/sw.js") {
      res.setHeader("Content-Type", "application/javascript");
      res.end(upgraded ? worker : legacy);
      return;
    }
    if (path === "/probe") {
      res.setHeader("Content-Type", "text/html");
      res.end("<html><body>Issue 299 worker upgrade probe</body></html>");
      return;
    }
    if (path.endsWith("/coach/reviews")) {
      res.setHeader("Content-Type", "application/json");
      res.end('{"reviews":[]}');
      return;
    }
    try {
      const file = resolve(dist, path.slice(1));
      if (!file.startsWith(dist + "/")) throw new Error("path");
      res.setHeader(
        "Content-Type",
        {
          ".js": "application/javascript",
          ".css": "text/css",
          ".html": "text/html",
          ".svg": "image/svg+xml",
          ".webmanifest": "application/manifest+json",
        }[extname(file)] ?? "application/octet-stream",
      );
      res.end(readFileSync(file));
    } catch {
      res.writeHead(404);
      res.end();
    }
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const origin = `http://127.0.0.1:${(server.address() as { port: number }).port}`;
  try {
    await page.goto(`${origin}/probe`);
    await page.evaluate(async () => {
      await navigator.serviceWorker.register("/sw.js");
      await navigator.serviceWorker.ready;
    });
    await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
    await page.evaluate(async () => {
      const cache = await caches.open("api-cache");
      await cache.put(
        "/api/v1/trades/fixture/coach/reviews",
        new Response("PRIVATE-LEGACY-MARKER", { headers: { "Content-Type": "application/json" } }),
      );
    });
    // Reproduce the old worker replay before replacing it.
    expect(
      await page.evaluate(async () => (await fetch("/api/v1/trades/fixture/coach/reviews")).text()),
    ).toBe("PRIVATE-LEGACY-MARKER");
    await page.evaluate(() => {
      (window as unknown as { changed: boolean }).changed = false;
      navigator.serviceWorker.addEventListener("controllerchange", () => {
        (window as unknown as { changed: boolean }).changed = true;
      });
    });
    upgraded = true;
    await page.evaluate(async () => {
      const reg = await navigator.serviceWorker.getRegistration();
      await reg!.update();
    });
    // No skip-waiting message, manual update approval or page close.
    await expect
      .poll(() => page.evaluate(() => (window as unknown as { changed: boolean }).changed))
      .toBe(true);
    await expect
      .poll(() =>
        page.evaluate(async () => {
          let count = 0;
          for (const name of await caches.keys()) {
            const cache = await caches.open(name);
            for (const req of await cache.keys()) {
              if (req.url.includes("/coach/reviews")) count++;
            }
          }
          return count;
        }),
      )
      .toBe(0);
    expect(
      await page.evaluate(async () => (await fetch("/api/v1/trades/fixture/coach/reviews")).text()),
    ).toBe('{"reviews":[]}');
    await context.setOffline(true);
    expect(
      await page.evaluate(async () => {
        try {
          return (await fetch("/api/v1/trades/fixture/coach/reviews")).text();
        } catch {
          return "NETWORK_ONLY_BLOCKED";
        }
      }),
    ).toBe("NETWORK_ONLY_BLOCKED");
  } finally {
    await context.setOffline(false);
    await new Promise<void>((resolve) => server.close(() => resolve()));
  }
});
