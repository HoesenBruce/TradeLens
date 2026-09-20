import { expect, test } from "@playwright/test";

// Exercise the live API, without the service worker serving cached responses.
test.use({ serviceWorkers: "block" });
const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";

test("news overview filters, persists, resets, paginates and handles loading/errors", async ({
  page,
  request,
}, info) => {
  const login = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(login.ok()).toBeTruthy();
  const auth = await login.json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  const ids: string[] = [];
  const prefix = `Overview ${Date.now()}`;
  try {
    for (let i = 0; i < 12; i++) {
      const response = await request.post(`${api}/news`, {
        headers,
        data: {
          title: `${prefix} ${i}`,
          source: "E2E",
          published_at: i === 0 ? "2026-09-20T12:00:00Z" : "2026-09-10T12:00:00Z",
          assets: [{ asset_type: "stock", symbol: i === 0 ? "285A" : "N225" }],
        },
      });
      expect(response.status()).toBe(201);
      const n = await response.json();
      ids.push(n.id);
      const prediction = await request.post(`${api}/news/${n.id}/predictions`, {
        headers,
        data: {
          news_asset_id: n.assets[0].id,
          direction: i === 0 ? "bullish" : "bearish",
          confidence: i === 0 ? 80 : null,
          reasoning: "E2E judgment",
          horizons: [1, 5],
        },
      });
      expect(prediction.status()).toBe(201);
    }
    await page.addInitScript(
      ({ token, api }) => {
        localStorage.setItem("tm_token", token);
        localStorage.setItem("tm_api_base", api);
      },
      { token: auth.access_token, api },
    );
    await page.goto(`${web}/news`);
    const reset = page.getByRole("button", { name: "Reset filters", exact: true }).first();
    await reset.click();
    await expect(page.getByRole("table", { name: "News theses" })).toBeVisible();
    await expect(page.getByRole("row").filter({ hasText: `${prefix} 0` })).toContainText("80%");
    await page.getByRole("button", { name: "Next page", exact: true }).click();
    await expect(page.getByRole("button", { name: "Previous page", exact: true })).toBeEnabled();
    await page.getByRole("button", { name: "Previous page", exact: true }).click();
    await page.getByRole("combobox", { name: "Direction filter" }).selectOption("bullish");
    await expect(page.locator("tbody > tr")).toHaveCount(1);
    await page.getByRole("combobox", { name: "Direction filter" }).selectOption("bearish");
    await expect(page.getByText(`${prefix} 0`, { exact: true })).toBeHidden();
    await page.getByRole("textbox", { name: "Symbol", exact: true }).fill("285a");
    await expect(page.getByText("No matching theses")).toBeVisible();
    await page.getByRole("combobox", { name: "Direction filter" }).selectOption("bullish");
    await expect(page.getByText(`${prefix} 0`, { exact: true })).toBeVisible();
    await page.reload();
    await expect(page.getByRole("textbox", { name: "Symbol", exact: true })).toHaveValue("285a");
    await expect(page.getByRole("combobox", { name: "Direction filter" })).toHaveValue("bullish");
    await page.goto(`${web}/home`);
    await page.goto(`${web}/news`);
    await expect(page.getByRole("combobox", { name: "Direction filter" })).toHaveValue("bullish");
    await reset.click();
    await page.getByLabel("From", { exact: true }).fill("2026-09-15");
    await expect(page.locator("tbody > tr")).toHaveCount(1);
    await page.getByLabel("To", { exact: true }).fill("2026-09-21");
    await page.screenshot({ path: info.outputPath("news-filtered.png"), fullPage: true });
    await reset.click();
    await page.getByLabel("To", { exact: true }).fill("2026-09-15");
    await expect(page.getByText(`${prefix} 0`, { exact: true })).toBeHidden();
    await reset.click();
    await page.getByRole("combobox", { name: "Validation filter" }).selectOption("validated");
    await expect(page.getByText("No matching theses")).toBeVisible();
    await page.getByRole("combobox", { name: "Validation filter" }).selectOption("pending");
    await expect(page.getByRole("table")).toBeVisible();
    await reset.click();
    await page.getByRole("combobox", { name: "Rows per page" }).selectOption("20");
    await expect(page.locator("tbody > tr")).toHaveCount(12);
    await page.screenshot({ path: info.outputPath("news-overview.png"), fullPage: true });
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route(`${api}/news`, async (route) => {
      await gate;
      await route.continue();
    });
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator('[data-slot="skeleton"]').first()).toBeVisible();
    release();
    await expect(page.getByRole("table")).toBeVisible();
    await page.unroute(`${api}/news`);
    await page.route(`${api}/news`, (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"message":"E2E failure"}',
      }),
    );
    await page.reload();
    await expect(page.getByText("Could not load news")).toBeVisible({ timeout: 15000 });
    await page.unroute(`${api}/news`);
    await page.getByRole("button", { name: "Try again" }).click();
    await expect(page.getByRole("table")).toBeVisible();
    for (const id of ids) await request.delete(`${api}/news/${id}`, { headers });
    await page.reload();
    await expect(page.getByText("No news theses yet")).toBeVisible();
  } finally {
    for (const id of ids) await request.delete(`${api}/news/${id}`, { headers });
  }
});
