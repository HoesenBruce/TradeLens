import { expect, test, type Page } from "@playwright/test";
import type { Trade } from "../src/lib/api/types";

// Run only with a disposable API. All accounts and executions below are synthetic.
test.use({ serviceWorkers: "block", hasTouch: true, viewport: { width: 1440, height: 1000 } });
const api = process.env.E2E_API_URL ?? "http://localhost:8092/api/v1";

async function explain(page: Page, metric: string, content: string, index = 0) {
  const trigger = page.locator(`[data-metric-help="${metric}"]`).nth(index);
  await trigger.press("Enter");
  const popup = page.locator('[data-slot="popover-popup"]');
  await expect(popup).toBeVisible();
  await expect(popup).toContainText(content);
  await expect(popup.locator("dt").first()).toHaveText("Computation");
  await popup.getByRole("button", { name: "Close explanation", exact: true }).click();
  await expect(popup).not.toBeVisible();
  await expect(trigger).toBeFocused();
}

test("definitions match real API trade states, lists, detail, reports and all five locales", async ({
  page,
  request,
}, info) => {
  test.setTimeout(180_000);
  const credentials = { email: "issue316@synthetic.invalid", password: "synthetic-316-password" };
  await request.post(`${api}/setup`, { data: { ...credentials, name: "Synthetic metrics QA" } });
  const login = await request.post(`${api}/auth/login`, { data: credentials });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const accountResponse = await request.post(`${api}/accounts`, {
    headers,
    data: {
      name: "Synthetic metric help 325",
      broker: "Manual",
      account_type: "margin",
      base_currency: "JPY",
      starting_balance: 10000,
    },
  });
  expect(accountResponse.status()).toBe(201);
  const { id: accountId } = await accountResponse.json();
  async function fill(
    symbol: string,
    side: string,
    quantity: number,
    price: number,
    executed_at: string,
    fees = 0,
    precision = "timestamp",
  ) {
    const response = await request.post(`${api}/executions`, {
      headers,
      data: {
        account_id: accountId,
        symbol,
        instrument_type: "stock",
        side,
        quantity,
        price,
        fees,
        executed_at,
        details: { source_time_precision: precision },
      },
    });
    expect(response.status(), await response.text()).toBe(201);
  }
  for (const [symbol, qty, exit, fee] of [
    ["METRICWIN", 100, 110, 1],
    ["METRICLOSS", 10, 90, 0],
    ["METRICFLAT", 10, 100, 0],
  ] as const) {
    await fill(symbol, "buy", qty, 100, "2026-10-13T01:00:00Z", fee);
    await fill(symbol, "sell", qty, exit, "2026-10-13T02:15:00Z", fee);
  }
  for (const [symbol, day] of [
    ["6991", "14"],
    ["6992", "13"],
  ]) {
    await fill(symbol, "buy", 10, 100, "2026-10-13T09:00:00+09:00", 0, "date");
    await fill(symbol, "sell", 10, 110, `2026-10-${day}T10:00:00+09:00`, 0, "date");
  }
  for (const [symbol, sides] of [
    ["METRICPART", ["buy", "sell"]],
    ["METRICSHORT", ["sell", "buy"]],
  ] as const) {
    await fill(symbol, sides[0], 100, 100, "2026-10-13T01:00:00Z");
    await fill(symbol, sides[1], 80, 110, "2026-10-13T02:15:00Z");
  }
  await fill("METRICREV", "buy", 5, 100, "2026-10-13T01:00:00Z");
  await fill("METRICREV", "sell", 7, 110, "2026-10-13T02:15:00Z");
  const readTrades = async (): Promise<Trade[]> =>
    (await request.get(`${api}/trades?account_id=${accountId}`, { headers })).json();
  const before = await readTrades();
  const trade = (symbol: string) => before.find((t) => t.symbol === symbol)!;
  expect(trade("METRICWIN")).toMatchObject({
    net_pnl: 998,
    gross_pnl: 1000,
    return_pct: 9.98,
    fees_total: 2,
  });
  expect(trade("METRICPART")).toMatchObject({
    net_pnl: null,
    quantities: { entry_qty: 100, exit_qty: 80, open_qty: 20, remaining_pct: 20 },
  });
  expect(trade("METRICSHORT").direction).toBe("short");
  expect(trade("6991").holding).toMatchObject({ kind: "calendar", calendar_days: 1 });
  expect(trade("6992").holding).toMatchObject({ kind: "calendar", calendar_days: 0 });
  expect(before.filter((t) => t.symbol === "METRICREV")).toHaveLength(2);
  const journal = await request.patch(`${api}/trades/${trade("METRICWIN").id}`, {
    headers,
    data: { initial_risk: 500, target_price: 120, stop_price: 90 },
  });
  expect(journal.ok()).toBeTruthy();
  const saved = await readTrades();
  const summary = await (
    await request.get(`${api}/analytics/summary?account_id=${accountId}`, { headers })
  ).json();
  expect(summary).toMatchObject({
    total_trades: 6,
    wins: 4,
    losses: 1,
    breakeven: 1,
    net_pnl: 1148,
    profit_factor: 12.48,
  });
  expect(summary.win_rate).toBeCloseTo(4 / 6);
  await page.addInitScript(
    ({ token, api, accountId }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      if (!localStorage.getItem("tm-locale")) localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [accountId] }, version: 1 }),
      );
    },
    { token, api, accountId },
  );
  await page.goto("/trades");
  await expect(page.getByText("METRICWIN", { exact: true })).toBeVisible();
  const initialSort = await page.locator("th[aria-sort]").allTextContents();
  for (const [id, text] of [
    ["holding", "same date = 0 days"],
    ["quantities", "Closed = opened"],
    ["pnl", "Default open trades"],
    ["tradeReturn", "unrounded weighted entry"],
    ["fees", "fees + commissions"],
    ["realizedR", "Net P&L ÷ initial risk"],
  ]) {
    await explain(page, id, text);
    await expect(page.locator('[role="dialog"]')).not.toBeVisible();
  }
  // The help button must not trigger column sorting or open a row.
  expect(await page.locator("th[aria-sort]").allTextContents()).toEqual(initialSort);
  const row = page
    .locator("tbody tr")
    .filter({ has: page.getByText("METRICWIN", { exact: true }) });
  await row.click();
  const sheet = page.getByRole("dialog");
  await expect(sheet).toBeVisible();
  await explain(page, "holding", "Precise endpoints", 1);
  await expect(sheet).toBeVisible();
  await explain(page, "plannedRR", "Long: (target − entry)");
  await expect(sheet).toBeVisible();
  await page.getByRole("button", { name: "Open full page", exact: true }).click();
  await expect(page.getByRole("heading", { name: "METRICWIN", exact: true })).toBeVisible();
  await expect(page.getByLabel(/1h 15m\. Elapsed time/)).toBeVisible();
  await explain(page, "plannedRR", "Short: (entry − target)");
  await explain(page, "realizedR", "Dividends are excluded");
  await expect(page.getByText("+9.98%", { exact: true })).toBeVisible();
  await page.reload();
  await explain(page, "pnl", "net = gross − fees");
  for (const symbol of ["6991", "6992", "METRICPART", "METRICSHORT"]) {
    await page.goto(`/trades/${trade(symbol).id}`);
    await explain(page, "holding", "SBI or CSV", 0);
    await explain(page, "quantities", "Long and short use positive magnitudes");
    if (symbol === "6991") await expect(page.getByLabel(/1 day\. Calendar days/)).toBeVisible();
    if (symbol === "6992") await expect(page.getByLabel(/0 days\. Calendar days/)).toBeVisible();
    if (symbol.startsWith("METRIC"))
      await expect(page.getByRole("progressbar").first()).toHaveAttribute("aria-valuenow", "80");
  }
  await page.goto("/reports");
  await explain(page, "winRate", "breakeven");
  await explain(page, "profitFactor", "API returns 0, not infinity");
  await explain(page, "payoff", "Median mode");
  await explain(page, "reportPnl", "auxiliary Gross row");
  await explain(page, "drawdown", "Withdrawals can create drawdown");
  const unit = page.getByRole("main").getByRole("group", { name: "Unit", exact: true });
  await unit.getByRole("button", { name: "%", exact: true }).click();
  await explain(page, "reportPnl", "positive signed cash-ledger total");
  await explain(page, "drawdown", "funding denominator");
  await unit.getByRole("button", { name: "$", exact: true }).click();
  await explain(page, "reportPnl", "Net = summary net P&L");
  await page.screenshot({ path: info.outputPath("reports-desktop.png"), animations: "disabled" });
  await explain(page, "periodReturns", "Annualized = total × 365");
  await page.getByRole("tab", { name: "Win / Loss", exact: true }).click();
  await explain(page, "rollingWinRate", "Fewer than N trades");
  await explain(page, "evolution", "Buckets use UTC dates");
  await page.getByRole("tab", { name: "Risk", exact: true }).click();
  await explain(page, "drawdownPct", "running visible peak");
  // Amount and percentage have independent explanations for each surface.
  await explain(page, "accountPerformance", "Other brokers: existing completed-group net P&L");
  await explain(page, "accountReturn", "existing funding denominator is unchanged");
  await page.goto(`/accounts/${accountId}`);
  await explain(page, "accountRealizedPnl", "full history is independent of header filters");
  await explain(page, "accountDetailReturn", "Hidden only when the denominator is zero");
  await expect(page.getByRole("main").getByText(/11\.48%/)).toBeVisible();
  for (const [locale, close] of [
    ["en", "Close explanation"],
    ["zh-CN", "关闭说明"],
    ["zh-HK", "關閉說明"],
    ["ja", "説明を閉じる"],
    ["ko", "설명 닫기"],
  ]) {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
    await page.goto(`/trades/${trade("6991").id}`);
    await page.reload();
    const trigger = page.locator('[data-metric-help="holding"]');
    await trigger.tap();
    const popup = page.locator('[data-slot="popover-popup"]');
    await expect(popup.getByRole("button", { name: close, exact: true })).toBeVisible();
    expect(await popup.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBeTruthy();
    const box = await popup.boundingBox();
    expect(box!.x + box!.width).toBeLessThanOrEqual(390);
    await page.screenshot({
      path: info.outputPath(`detail-${locale}-narrow.png`),
      animations: "disabled",
    });
    await popup.getByRole("button", { name: close, exact: true }).click();
    await expect(popup).not.toBeVisible();
    const qty = page.locator('[data-metric-help="quantities"]');
    await qty.tap();
    const quantityTitles: Record<string, string> = {
      en: "Opened, closed and remaining quantity",
      "zh-CN": "开仓、平仓与剩余数量",
      "zh-HK": "開倉、平倉與剩餘數量",
      ja: "新規・決済・残数量",
      ko: "진입·청산·잔여 수량",
    };
    await expect(popup).toHaveAccessibleName(quantityTitles[locale]);
    await popup.getByRole("button", { name: close, exact: true }).click();
    await page.goto("/reports?tab=overview");
    const winRate = page.getByRole("main").locator('[data-metric-help="winRate"]').first();
    await winRate.tap();
    const winTitles: Record<string, string> = {
      en: "Win rate",
      "zh-CN": "胜率",
      "zh-HK": "勝率",
      ja: "勝率",
      ko: "승률",
    };
    await expect(popup).toHaveAccessibleName(winTitles[locale]);
    const reportBox = await popup.boundingBox();
    expect(reportBox!.x).toBeGreaterThanOrEqual(0);
    expect(reportBox!.x + reportBox!.width).toBeLessThanOrEqual(390);
    await page.screenshot({
      path: info.outputPath(`reports-${locale}-narrow.png`),
      animations: "disabled",
    });
    await popup.getByRole("button", { name: close, exact: true }).click();
  }
  await page.evaluate(() => localStorage.setItem("tm-locale", "en"));
  await page.goto("/trades");
  await page.reload();
  await page.locator('[data-metric-help="quantities"]').first().tap();
  await expect(page.locator('[data-slot="popover-popup"]')).toContainText(
    "not current market value",
  );
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).not.toBeVisible();
  await page.screenshot({ path: info.outputPath("list-narrow.png"), animations: "disabled" });
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
  ).toBeTruthy();
  const stable = (rows: Trade[]) =>
    rows.map((row) => ({
      ...row,
      holding: row.holding?.ongoing ? { ...row.holding, seconds: null } : row.holding,
    }));
  expect(stable(await readTrades())).toEqual(stable(saved));
});

test("zero funding and no losses keep their existing boundaries, including simulation drawdown", async ({
  page,
  request,
}, info) => {
  test.setTimeout(90_000);
  const credentials = { email: "issue316@synthetic.invalid", password: "synthetic-316-password" };
  await request.post(`${api}/setup`, { data: { ...credentials, name: "Synthetic metrics QA" } });
  const login = await request.post(`${api}/auth/login`, { data: credentials });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const response = await request.post(`${api}/accounts`, {
    headers,
    data: {
      name: "Synthetic metric boundaries 325",
      broker: "Manual",
      account_type: "margin",
      base_currency: "JPY",
      starting_balance: 0,
    },
  });
  expect(response.status()).toBe(201);
  const { id: accountId } = await response.json();
  for (let i = 0; i < 12; i++) {
    for (const [side, price, hour] of [
      ["buy", 100, "01"],
      ["sell", 110, "02"],
    ] as const) {
      const fill = await request.post(`${api}/executions`, {
        headers,
        data: {
          account_id: accountId,
          symbol: `METRICZERO${i}`,
          instrument_type: "stock",
          side,
          quantity: 10,
          price,
          executed_at: `2026-10-13T${hour}:00:00Z`,
        },
      });
      expect(fill.status()).toBe(201);
    }
  }
  const summary = await (
    await request.get(`${api}/analytics/summary?account_id=${accountId}`, { headers })
  ).json();
  expect(summary).toMatchObject({
    total_trades: 12,
    wins: 12,
    losses: 0,
    profit_factor: 0,
    net_pnl: 1200,
    win_rate: 1,
  });
  await page.addInitScript(
    ({ token, api, accountId }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [accountId] }, version: 1 }),
      );
    },
    { token, api, accountId },
  );
  await page.goto("/reports");
  const main = page.getByRole("main");
  const pf = main.locator('[data-metric-help="profitFactor"]').first();
  await expect(pf.locator("..").locator("..")).toContainText("—");
  await expect(
    main
      .getByRole("group", { name: "Unit", exact: true })
      .getByRole("button", { name: "%", exact: true }),
  ).toHaveCount(0);
  await explain(page, "payoff", "No losses but a positive win displays ∞");
  await expect(main.getByText("∞", { exact: true })).toBeVisible();
  await explain(page, "profitFactor", "no losses the API returns 0");
  await page.getByRole("tab", { name: "Risk", exact: true }).click();
  await explain(page, "simulationDrawdown", "not account equity");
  await page.screenshot({
    path: info.outputPath("zero-loss-simulation.png"),
    animations: "disabled",
  });
});
