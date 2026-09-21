import { createServer } from "node:http";
import { expect, test } from "@playwright/test";
test.use({ serviceWorkers: "block" });
const api = process.env.E2E_API_URL ?? "http://localhost:8096/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5186";
test("AI review through real API", async ({ page, request }, info) => {
  test.setTimeout(120000);
  const login = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(login.ok()).toBeTruthy();
  const auth = await login.json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  let fail = false;
  let empty = false;
  const asset = {
    asset_type: "stock",
    symbol: "285A",
    market: "JP",
    exchange: "TSE",
    display_name: "",
    direction: "bullish",
    confidence: 70,
    reasoning: "AI demand thesis",
    catalysts: "Orders",
    risks: "Execution",
    horizons: [1, 5],
  };
  const provider = createServer(async (req, res) => {
    for await (const _chunk of req) {
      /* consume request */
    }
    await new Promise((r) => setTimeout(r, 400));
    res.setHeader("Content-Type", "application/json");
    if (fail) {
      res.writeHead(503);
      res.end("{}");
      return;
    }
    res.end(
      JSON.stringify({
        choices: [
          {
            message: {
              content: JSON.stringify({
                summary: "AI summary",
                category: "Industry",
                assets: empty ? [] : [asset, { ...asset, symbol: "AAPL", market: "US" }],
              }),
            },
          },
        ],
      }),
    );
  });
  await new Promise<void>((resolve) => provider.listen(0, "127.0.0.1", resolve));
  const { port } = provider.address() as { port: number };
  expect(
    (
      await request.put(`${api}/settings/coach`, {
        headers,
        data: {
          enabled: true,
          base_url: `http://127.0.0.1:${port}`,
          model: "fixture",
          api_key: "fixture",
        },
      })
    ).ok(),
  ).toBeTruthy();
  const created = await request.post(`${api}/news`, {
    headers,
    data: {
      title: `AI review ${Date.now()}`,
      source: "Manual",
      published_at: "2026-09-21T00:00:00Z",
      original_text: "285A supply announcement",
      summary: "Manual summary",
      category: "Manual",
      assets: [{ asset_type: "stock", symbol: "285A", market: "JP" }],
    },
  });
  expect(created.status()).toBe(201);
  const news = await created.json();
  await request.post(`${api}/news/${news.id}/predictions`, {
    headers,
    data: {
      news_asset_id: news.assets[0].id,
      direction: "neutral",
      confidence: 45,
      reasoning: "Original user prediction",
      horizons: [1],
    },
  });
  await page.addInitScript(
    ({ token, api }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
    },
    { token: auth.access_token, api },
  );
  const read = async () => (await request.get(`${api}/news/${news.id}`, { headers })).json();
  const baseline = await read();
  try {
    await page.goto(`${web}/news/${news.id}`);
    await page.getByRole("button", { name: "Analyze with AI" }).click();
    const d = page.getByRole("dialog");
    await expect(d.getByRole("status")).toContainText("Analyzing");
    await expect(d.getByLabel("Suggested summary")).toHaveValue("AI summary");
    await expect(d.getByRole("button", { name: "Save selected suggestions" })).toBeDisabled();
    await d.getByLabel("Suggested summary").fill("Abandoned");
    await d.getByRole("checkbox", { name: "Accept summary" }).check();
    await d.getByRole("button", { name: "Cancel", exact: true }).click();
    expect(await read()).toEqual(baseline);
    await page.getByRole("button", { name: "Analyze with AI" }).click();
    await expect(d.getByLabel("Suggested summary")).toHaveValue("AI summary");
    await expect(d.getByRole("checkbox", { name: "Accept summary" })).not.toBeChecked();
    await d.getByRole("checkbox", { name: "Accept summary" }).check();
    await d.getByRole("checkbox", { name: "Accept summary" }).uncheck();
    await d.getByRole("checkbox", { name: "Accept summary" }).check();
    await d.getByLabel("Suggested summary").fill("Reviewed summary");
    await d.getByRole("checkbox", { name: "Accept category" }).check();
    await d.getByRole("checkbox", { name: "Accept category" }).uncheck();
    const first = d.getByRole("region", { name: "Suggestion 1", exact: true });
    await first.getByRole("checkbox", { name: "Accept asset" }).check();
    await first.getByRole("checkbox", { name: "Accept asset" }).uncheck();
    await first.getByRole("checkbox", { name: "Accept asset" }).check();
    await first.getByRole("checkbox", { name: "Include prediction" }).uncheck();
    await expect(first.getByLabel("Direction")).toHaveCount(0);
    await first.getByRole("checkbox", { name: "Include prediction" }).check();
    await first.getByLabel("Direction").selectOption("bearish");
    await first.getByLabel("reasoning", { exact: true }).fill("Reviewed AI risk");
    await first.getByRole("checkbox", { name: "1D", exact: true }).uncheck();
    await first.getByRole("checkbox", { name: "5D", exact: true }).uncheck();
    await d.getByRole("button", { name: "Save selected suggestions" }).click();
    await expect(d.getByRole("alert")).toContainText("at least one horizon");
    await first.getByRole("checkbox", { name: "3D", exact: true }).check();
    await d
      .getByRole("region", { name: "Suggestion 2", exact: true })
      .getByRole("button", { name: "Reject suggestion" })
      .click();
    await d.getByRole("button", { name: "Add manual asset" }).click();
    const second = d.getByRole("region", { name: "Suggestion 2", exact: true });
    await second.getByLabel("symbol", { exact: true }).fill("1306");
    await second.getByLabel("Type").selectOption("etf");
    await second.getByLabel("market", { exact: true }).fill("JP");
    await second.getByLabel("reasoning", { exact: true }).fill("Manual ETF thesis");
    await page.screenshot({ path: info.outputPath("review.png"), fullPage: true });
    await d.getByRole("button", { name: "Save selected suggestions" }).click();
    await expect(d).toHaveCount(0);
    await page.reload();
    await expect(page.getByText("Reviewed summary", { exact: true })).toBeVisible();
    await expect(page.getByText("Reviewed AI risk", { exact: true })).toBeVisible();
    const saved = await read();
    expect(saved.category).toBe("Manual");
    expect(saved.assets).toHaveLength(3);
    expect(saved.predictions).toHaveLength(3);
    expect(
      saved.predictions.find((p: { reasoning: string }) => p.reasoning === "Reviewed AI risk"),
    ).toMatchObject({ source: "ai", direction: "bearish", horizons: [3] });
    expect(
      saved.predictions.find((p: { reasoning: string }) => p.reasoning === "Manual ETF thesis"),
    ).toMatchObject({ source: "user" });
    expect(saved.predictions).toContainEqual(baseline.predictions[0]);
    await page.screenshot({ path: info.outputPath("accepted.png"), fullPage: true });
    fail = true;
    await page.getByRole("button", { name: "Analyze with AI" }).click();
    await expect(d.getByRole("alert")).toContainText("No data was changed");
    expect(await read()).toEqual(saved);
    await page.screenshot({ path: info.outputPath("provider-error.png"), fullPage: true });
    fail = false;
    empty = true;
    await d.getByRole("button", { name: "Retry analysis" }).click();
    await expect(d.getByText("No asset suggestions. You can add a manual asset.")).toBeVisible();
    await d.getByRole("button", { name: "Cancel", exact: true }).click();
    expect(await read()).toEqual(saved);
  } finally {
    await new Promise<void>((resolve, reject) =>
      provider.close((err) => (err ? reject(err) : resolve())),
    );
  }
});
