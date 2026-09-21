import { createServer } from "node:http";
import { execFileSync } from "node:child_process";
import { expect, test } from "@playwright/test";

test.use({ serviceWorkers: "block" });
const api = process.env.E2E_API_URL ?? "http://localhost:8097/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5186";
const sqlite = process.env.E2E_SQLITE_PATH;

test("performance uses persisted validations and server filters", async ({
  page,
  request,
}, info) => {
  test.skip(
    !sqlite,
    "Requires an explicit disposable SQLite database and API HTTP provider on port 18963.",
  );
  test.setTimeout(120000);
  const login = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(login.ok()).toBeTruthy();
  const auth = await login.json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  let marketCalls = 0;
  const provider = createServer((req, res) => {
    marketCalls++;
    const url = new URL(req.url!, "http://localhost");
    const symbol = url.searchParams.get("symbol")!;
    res.setHeader("Content-Type", "application/json");
    if (symbol.includes("ERR")) {
      res.writeHead(503);
      res.end("{}");
      return;
    }
    const from = new Date(url.searchParams.get("from")!);
    const to = new Date(url.searchParams.get("to")!);
    const bars = [];
    for (let d = from.getTime(); d < to.getTime(); d += 86400000) {
      const tokyo = new Date(d + 9 * 3600000);
      if ([0, 6].includes(tokyo.getUTCDay())) continue;
      bars.push({
        timestamp: new Date(d).toISOString(),
        market_date: tokyo.toISOString().slice(0, 10),
        open: 100,
        high: 104,
        low: 99,
        close: 102,
        volume: 1000,
      });
    }
    res.end(
      JSON.stringify({
        symbol,
        interval: "D",
        timezone: "Asia/Tokyo",
        source: "E2E archive",
        adjustment_status: "unadjusted",
        fetched_at: new Date().toISOString(),
        bars,
      }),
    );
  });
  await new Promise<void>((resolve) => provider.listen(18963, "127.0.0.1", resolve));
  try {
    const seed = async (
      symbol: string,
      source: string,
      category: string,
      date: string,
      horizons: number[],
      evaluate: boolean,
    ) => {
      const created = await request.post(`${api}/news`, {
        headers,
        data: {
          title: `Performance ${symbol}`,
          source: "Fixture",
          published_at: date,
          category,
          assets: [{ asset_type: "stock", symbol, market: "JP" }],
        },
      });
      expect(created.status()).toBe(201);
      const n = await created.json();
      const prediction = await request.post(`${api}/news/${n.id}/predictions`, {
        headers,
        data: {
          news_asset_id: n.assets[0].id,
          direction: source === "ai" ? "bearish" : "bullish",
          confidence: 60,
          reasoning: "Fixture",
          horizons,
        },
      });
      expect(prediction.status()).toBe(201);
      const p = await prediction.json();
      // Backdate only the explicitly chosen disposable fixture DB; evaluations still go through the real API/engine.
      execFileSync("python3", [
        "-c",
        "import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); c.execute(\"UPDATE predictions SET source=?,created_at='2026-09-13 00:00:00+00:00',updated_at='2026-09-13 00:00:00+00:00' WHERE id=?\",(sys.argv[2],sys.argv[3])); c.commit()",
        sqlite!,
        source,
        p.id,
      ]);
      if (evaluate) {
        const validated = await request.post(`${api}/news/${n.id}/predictions/${p.id}/validate`, {
          headers,
        });
        expect(validated.ok()).toBeTruthy();
        const rows = await validated.json();
        expect(rows[0].result.status).toBe(symbol === "ERR" ? "unavailable" : "validated");
      }
    };
    // Use a fresh database for each run so the aggregate assertions remain sharp.
    const initial = await request.get(`${api}/news/performance`, { headers });
    expect((await initial.json()).counts.total).toBe(0);
    await seed("285A", "user", "Industry", "2026-09-01T23:00:00Z", [1, 20], true);
    await seed("5801", "ai", "Macro", "2026-09-02T00:00:00Z", [1], true);
    await seed("ERR", "user", "Data", "2026-09-03T00:00:00Z", [1], true);
    await seed("PEND", "user", "Future", "2026-09-03T00:00:00Z", [1], false);
    const calls = marketCalls;
    await page.addInitScript(
      ({ token, api }) => {
        localStorage.setItem("tm_token", token);
        localStorage.setItem("tm_api_base", api);
      },
      { token: auth.access_token, api },
    );
    let release!: () => void;
    const gate = new Promise<void>((r) => {
      release = r;
    });
    await page.route(`${api}/news/performance*`, async (route) => {
      await gate;
      await route.continue();
    });
    await page.goto(`${web}/news/performance`, { waitUntil: "domcontentloaded" });
    await expect(page.getByRole("status", { name: "Loading performance" })).toBeVisible();
    release();
    const counts = page.getByRole("region", { name: "Performance counts" });
    const count = async (label: string, value: number) =>
      expect(
        counts
          .getByText(label, { exact: true })
          .locator("..")
          .getByText(String(value), { exact: true }),
      ).toBeVisible();
    await count("total", 5);
    await count("pending", 2);
    await count("validated", 2);
    await count("unavailable", 1);
    await page.unroute(`${api}/news/performance*`);
    const sources = page.getByRole("region", { name: "AI / User", exact: true });
    await expect(sources).toContainText("100.0% · n=1 (1 correct)");
    await expect(sources).toContainText("0.0% · n=1 (0 correct)");
    await expect(page.getByRole("region", { name: "By horizon", exact: true })).toContainText(
      "Not available · n=0",
    );
    await page.screenshot({ path: info.outputPath("overview.png"), fullPage: true });
    const filters = page.getByRole("region", { name: "Performance filters" });
    await filters.getByLabel("Source", { exact: true }).selectOption("ai");
    await count("total", 1);
    await expect(sources).not.toContainText("100.0%");
    await page.reload();
    await expect(filters.getByLabel("Source", { exact: true })).toHaveValue("ai");
    await count("total", 1);
    await page.getByRole("link", { name: "Back to news" }).click();
    await page.goBack();
    await expect(filters.getByLabel("Source", { exact: true })).toHaveValue("ai");
    await count("total", 1);
    await filters.getByRole("button", { name: "Reset filters" }).click();
    await count("total", 5);
    await expect(filters.getByLabel("Source", { exact: true })).toHaveValue("");
    await filters.getByLabel("Horizon", { exact: true }).selectOption("20");
    await count("total", 1);
    await count("pending", 1);
    await count("validated", 0);
    await expect(sources).toContainText("Not available · n=0");
    await expect(sources).not.toContainText("%");
    await page.screenshot({ path: info.outputPath("pending.png"), fullPage: true });
    await filters.getByRole("button", { name: "Reset filters" }).click();
    await count("total", 5);
    await filters.getByLabel("Symbol", { exact: true }).fill("285A");
    await count("total", 2);
    await expect(page.getByRole("region", { name: "By asset", exact: true })).not.toContainText(
      "5801",
    );
    await filters.getByLabel("Category", { exact: true }).fill("Macro");
    await expect(page.getByText("No matching predictions", { exact: true })).toBeVisible();
    await count("total", 0);
    await page.screenshot({ path: info.outputPath("empty.png"), fullPage: true });
    await filters.getByRole("button", { name: "Reset filters" }).click();
    await count("total", 5);
    await filters.getByLabel("Asset type", { exact: true }).selectOption("etf");
    await expect(page.getByText("No matching predictions", { exact: true })).toBeVisible();
    await filters.getByLabel("Asset type", { exact: true }).selectOption("stock");
    await count("total", 5);
    await filters.getByLabel("Category", { exact: true }).fill("Macro");
    await count("total", 1);
    await filters.getByRole("button", { name: "Reset filters" }).click();
    await count("total", 5);
    await filters.getByLabel("Published from (UTC)").fill("2026-09-02");
    await filters.getByLabel("Published to (UTC)").fill("2026-09-02");
    await count("total", 1);
    await expect(sources).toContainText("0.0% · n=1");
    await page.reload();
    await expect(filters.getByLabel("Published from (UTC)")).toHaveValue("2026-09-02");
    await count("total", 1);
    await filters.getByRole("button", { name: "Reset filters" }).click();
    await count("total", 5);
    await page.route(`${api}/news/performance*`, (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"error":{"message":"Fixture failure"}}',
      }),
    );
    await page.reload();
    await expect(page.getByText("Could not load performance", { exact: true })).toBeVisible({
      timeout: 15000,
    });
    await page.screenshot({ path: info.outputPath("error.png"), fullPage: true });
    await page.unroute(`${api}/news/performance*`);
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await count("total", 5);
    await page.getByRole("link", { name: "Back to news" }).click();
    await page.getByRole("link", { name: "Performance report", exact: true }).click();
    await count("total", 5);
    expect(marketCalls).toBe(calls); // Report reads never fetch prices or recalculate outcomes.
  } finally {
    await new Promise<void>((resolve, reject) =>
      provider.close((err) => (err ? reject(err) : resolve())),
    );
  }
});
