import { execFileSync } from "node:child_process";
import { expect, test } from "@playwright/test";

test.use({ serviceWorkers: "block" });
const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";

test("direct news detail, User/AI hierarchy, editing, and deleted/missing records", async ({
  page,
  request,
}, info) => {
  const login = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(login.ok()).toBeTruthy();
  const auth = await login.json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  const title = `Detail ${Date.now()}`;
  const result = await request.post(`${api}/news`, {
    headers,
    data: {
      title,
      source: "Company filing",
      url: "https://example.com/filing",
      published_at: "2026-09-20T12:00:00Z",
      original_text: "Original announcement",
      summary: "Supply agreement",
      notes: "Watch execution",
      tags: ["Japan"],
      assets: [
        { asset_type: "stock", symbol: "285A", market: "JP", relation: "supplier" },
        { asset_type: "index", symbol: "N225", relation: "benchmark" },
      ],
    },
  });
  expect(result.status()).toBe(201);
  const news = await result.json();
  const asset = news.assets.find((a: { symbol: string }) => a.symbol === "285A");
  const prediction = await request.post(`${api}/news/${news.id}/predictions`, {
    headers,
    data: {
      news_asset_id: asset.id,
      direction: "bullish",
      confidence: 75,
      reasoning: "User demand thesis",
      catalysts: "Contract launch",
      risks: "Delivery delays",
      invalidation: "Contract cancelled",
      horizons: [1, 5],
    },
  });
  expect(prediction.status()).toBe(201);
  // Optional fixture injection into the explicitly selected disposable SQLite database; no AI endpoint is exposed.
  const sqlite = process.env.E2E_SQLITE_PATH;
  if (sqlite)
    execFileSync("python3", [
      "-c",
      "import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); c.execute(\"INSERT INTO predictions(id,news_asset_id,source,direction,reasoning) VALUES (?,?,'ai','bearish','AI alternative thesis')\",(sys.argv[2],sys.argv[3])); c.execute('INSERT INTO prediction_horizons VALUES (?,3)',(sys.argv[2],)); c.commit()",
      sqlite,
      `ai-${news.id}`,
      asset.id,
    ]);
  await page.addInitScript(
    ({ token, api }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
    },
    { token: auth.access_token, api },
  );
  try {
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route(`${api}/news/${news.id}`, async (route) => {
      await gate;
      await route.continue();
    });
    await page.goto(`${web}/news/${news.id}`, { waitUntil: "domcontentloaded" });
    await expect(page.locator('[data-slot="skeleton"]').first()).toBeVisible();
    release();
    await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible();
    await page.unroute(`${api}/news/${news.id}`);
    await expect(page.getByText("Original announcement", { exact: true })).toBeVisible();
    await expect(page.getByText("Contract cancelled", { exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: "Open source" })).toHaveAttribute(
      "href",
      "https://example.com/filing",
    );
    await expect(
      page.getByRole("table", { name: "Validation horizons for user prediction" }),
    ).toContainText("Pending validation");
    if (sqlite) {
      const ai = page.getByRole("article", { name: `AI prediction ai-${news.id}`, exact: true });
      await expect(ai.getByText("AI alternative thesis")).toBeVisible();
      await expect(ai.getByRole("button", { name: "Edit prediction" })).toHaveCount(0);
      await expect(ai.getByRole("table")).toContainText("3D");
      await ai.scrollIntoViewIfNeeded();
      await page.screenshot({ path: info.outputPath("news-detail-ai.png"), fullPage: true });
    }
    await page.getByRole("heading", { name: title, exact: true }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: info.outputPath("news-detail.png"), fullPage: true });
    await page.getByRole("button", { name: "Edit news thesis", exact: true }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("textbox", { name: "Title", exact: true }).fill("Abandoned");
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await page.getByRole("button", { name: "Edit news thesis", exact: true }).click();
    await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toHaveValue(title);
    await dialog.getByRole("textbox", { name: "Title", exact: true }).fill(`${title} updated`);
    await dialog.getByRole("button", { name: "Save changes" }).click();
    await expect(
      page.getByRole("heading", { name: `${title} updated`, exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Edit prediction", exact: true }).click();
    await dialog.getByRole("textbox", { name: "Risks", exact: true }).fill("Updated delivery risk");
    await dialog.getByRole("button", { name: "Save prediction" }).click();
    await expect(page.getByText("Updated delivery risk", { exact: true })).toBeVisible();
    await page.reload();
    await expect(page.getByText("Updated delivery risk", { exact: true })).toBeVisible();
    const indexAsset = page
      .getByRole("heading", { name: "N225", exact: true })
      .locator("xpath=ancestor::section[1]");
    await indexAsset.getByRole("button", { name: "Add prediction" }).click();
    await dialog
      .getByRole("textbox", { name: "Reasoning", exact: true })
      .fill("Index neutral view");
    await dialog.getByRole("button", { name: "Save prediction" }).click();
    await expect(indexAsset.getByText("Index neutral view")).toBeVisible();
    await indexAsset.getByRole("button", { name: "Delete prediction" }).click();
    await dialog.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(indexAsset.getByText("No predictions yet.")).toBeVisible();
    await page.route(`${api}/news/${news.id}`, (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"error":{"message":"E2E failure"}}',
      }),
    );
    await page.reload();
    await expect(page.getByText("Could not load news thesis", { exact: true })).toBeVisible({
      timeout: 15000,
    });
    await page.unroute(`${api}/news/${news.id}`);
    await page.getByRole("button", { name: "Try again" }).click();
    await expect(
      page.getByRole("heading", { name: `${title} updated`, exact: true }),
    ).toBeVisible();
    await page.getByRole("link", { name: "Back to news" }).click();
    await page.getByRole("button", { name: "Reset filters", exact: true }).first().click();
    await page.getByRole("link", { name: `${title} updated`, exact: true }).click();
    await expect(page).toHaveURL(`${web}/news/${news.id}`);
    await request.delete(`${api}/news/${news.id}`, { headers });
    await page.reload();
    await expect(page.getByText("News thesis not found", { exact: true })).toBeVisible();
    await page.screenshot({ path: info.outputPath("news-deleted.png"), fullPage: true });
    await page.goto(`${web}/news/missing-record`);
    await expect(page.getByText("News thesis not found", { exact: true })).toBeVisible();
  } finally {
    await request.delete(`${api}/news/${news.id}`, { headers });
  }
});
