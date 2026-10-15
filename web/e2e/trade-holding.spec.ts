import { expect, test } from "@playwright/test";
import type { Trade, TradeDetail } from "../src/lib/api/types";

test.use({
  serviceWorkers: "block",
  timezoneId: "America/Los_Angeles",
  viewport: { width: 1440, height: 1000 },
});
const api = process.env.E2E_API_URL ?? "http://localhost:8092/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5174";

test("holding precision survives imports, API readback, details, locales and viewer timezones", async ({
  page,
  request,
}, info) => {
  test.setTimeout(120_000);
  const credentials = {
    email: process.env.E2E_EMAIL ?? "issue316@synthetic.invalid",
    password: process.env.E2E_PASSWORD ?? "synthetic-316-password",
  };
  await request.post(`${api}/setup`, { data: { ...credentials, name: "Synthetic holding QA" } });
  const login = await request.post(`${api}/auth/login`, { data: credentials });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const response = await request.post(`${api}/accounts`, {
    headers,
    data: {
      name: "Synthetic 316",
      broker: "Manual",
      account_type: "margin",
      base_currency: "JPY",
      starting_balance: 0,
    },
  });
  expect(response.status()).toBe(201);
  const { id: accountId } = await response.json();
  const fill = async (symbol: string, side: string, executed_at: string, precision = "") => {
    const r = await request.post(`${api}/executions`, {
      headers,
      data: {
        account_id: accountId,
        symbol,
        instrument_type: "stock",
        side,
        quantity: 100,
        price: 1000,
        fees: 1,
        executed_at,
        details: { source_time_precision: precision },
      },
    });
    expect(r.status(), await r.text()).toBe(201);
  };
  await fill("E2EPRECISE", "buy", "2026-10-13T09:30:00-04:00");
  await fill("E2EPRECISE", "sell", "2026-10-13T10:45:00-04:00");
  await fill("6990", "buy", "2026-10-13T14:55:00Z", "date");
  await fill("6990", "sell", "2026-10-13T15:05:00Z");
  const importCSV = async (text: string, mapping?: Record<string, string>) => {
    const file = { name: "synthetic-316.csv", mimeType: "text/csv", buffer: Buffer.from(text) };
    if (!mapping) {
      const preview = await request.post(`${api}/imports`, {
        headers,
        multipart: { account_id: accountId, file },
      });
      expect(preview.ok(), await preview.text()).toBeTruthy();
      mapping = (await preview.json()).suggested_mapping;
    }
    const commit = await request.post(`${api}/imports/commit`, {
      headers,
      multipart: { account_id: accountId, file, column_mapping: JSON.stringify(mapping) },
    });
    expect(commit.ok(), await commit.text()).toBeTruthy();
  };
  await importCSV(
    "約定履歴照会\n約定日,銘柄,銘柄コード,市場,取引,約定数量,約定単価,手数料/諸経費等,受渡金額/決済損益\n" +
      [
        "2026/10/13,Synthetic SAME,6921,東証,現物買,100,1000,--,--",
        "2026/10/13,Synthetic SAME,6921,東証,現物売,100,1100,--,--",
        "2026/10/13,Synthetic NEXT,6922,東証,信用新規買,100,1000,--,--",
        "2026/10/14,Synthetic NEXT,6922,東証,信用返済売,100,1100,--,10000",
        "2026/10/13,Synthetic THREE,6920,東証,信用新規買,100,1000,--,--",
        "2026/10/16,Synthetic THREE,6920,東証,信用返済売,100,1100,--,10000",
        "2026/10/13,Synthetic OPEN,6923,東証,信用新規売,100,1000,--,--",
        "2026/10/13,Synthetic MISSING,6924,東証,現物売,100,1000,--,--",
      ].join("\n") +
      "\n",
  );
  await importCSV(
    "Symbol,Side,Qty,Price,Time\nE2EGEN,Buy,100,100,2026-10-13\nE2EGEN,Sell,100,110,2026-10-14\n",
    { symbol: "Symbol", side: "Side", quantity: "Qty", price: "Price", executed_at: "Time" },
  );
  const savedResponse = await request.get(`${api}/trades?account_id=${accountId}`, { headers });
  expect(savedResponse.ok()).toBeTruthy();
  const saved: Trade[] = await savedResponse.json();
  const bySymbol = (symbol: string) => saved.find((tr) => tr.symbol === symbol)!;
  expect(bySymbol("E2EPRECISE").holding).toMatchObject({ kind: "elapsed", seconds: 4500 });
  for (const [symbol, days] of [
    ["6921", 0],
    ["6922", 1],
    ["6920", 3],
    ["6990", 1],
    ["E2EGEN", 1],
  ] as const) {
    expect(bySymbol(symbol).holding).toMatchObject({
      kind: "calendar",
      calendar_days: days,
      seconds: null,
    });
    const detailResponse = await request.get(`${api}/trades/${bySymbol(symbol).id}`, { headers });
    const detail: TradeDetail = await detailResponse.json();
    expect(detail.holding).toEqual(bySymbol(symbol).holding);
    expect(detail.fills.some((f) => f.details?.source_time_precision === "date")).toBeTruthy();
  }
  expect(bySymbol("6924").holding?.kind).toBe("unknown");
  const todayTokyo = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Tokyo",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
  const ongoingDays = (Date.parse(todayTokyo) - Date.parse("2026-10-13")) / 86400000;
  expect(bySymbol("6923").holding).toMatchObject({
    kind: "calendar",
    ongoing: true,
    time_zone: "Asia/Tokyo",
    calendar_days: ongoingDays,
  });
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
  await page.goto(`${web}/trades`);
  const row = (symbol: string) =>
    page.locator("tbody tr").filter({ has: page.getByText(symbol, { exact: true }) });
  await expect(row("6921").getByLabel(/0 days\. Calendar days in Asia\/Tokyo/)).toBeVisible();
  await expect(row("6922").getByLabel(/1 day\. Calendar days/)).toBeVisible();
  await expect(row("E2EPRECISE").getByLabel(/1h 15m\. Elapsed time/)).toBeVisible();
  await page.screenshot({ animations: "disabled", path: info.outputPath("holding-list.png") });
  await row("6920").click();
  await expect(page.getByRole("dialog").getByLabel(/3 days\. Calendar days/)).toBeVisible();
  await expect(page.getByRole("alert")).toContainText("exact elapsed holding time");
  await page.screenshot({ animations: "disabled", path: info.outputPath("holding-peek.png") });
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).not.toBeVisible();
  await row("6920").click();
  await expect(page.getByRole("dialog").getByLabel(/3 days\. Calendar days/)).toBeVisible();
  await page.getByRole("button", { name: "Open full page", exact: true }).click();
  await expect(page.getByLabel(/3 days\. Calendar days/)).toBeVisible();
  await page.reload();
  await expect(page.getByLabel(/3 days\. Calendar days/)).toBeVisible();
  for (const [locale, expected] of [
    ["zh-CN", "3天"],
    ["ja", "3 日"],
    ["en", "3 days"],
  ]) {
    await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
    await page.reload();
    await expect(page.getByText(expected, { exact: true })).toBeVisible();
    await page.screenshot({
      animations: "disabled",
      path: info.outputPath(`holding-detail-${locale}.png`),
    });
  }
  await page.goto(`${web}/trades/${bySymbol("6923").id}`);
  await expect(
    page.getByLabel(new RegExp(`${ongoingDays} days? · ongoing\\. Calendar days`)),
  ).toBeVisible();
  await page.goto(`${web}/trades/${bySymbol("6924").id}`);
  await expect(page.getByLabel("—. Holding duration is unknown.")).toBeVisible();
  await page.goto(`${web}/trades/${bySymbol("E2EPRECISE").id}`);
  await expect(page.getByLabel(/1h 15m\. Elapsed time/)).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${web}/trades/${bySymbol("6921").id}`);
  await expect(page.getByLabel(/0 days\. Calendar days/)).toBeVisible();
  await page.screenshot({ animations: "disabled", path: info.outputPath("holding-compact.png") });
  await page.goto(`${web}/trades`);
  await expect(page.getByLabel(/0 days\. Calendar days in Asia\/Tokyo/)).toBeVisible();
  await page.screenshot({
    animations: "disabled",
    path: info.outputPath("holding-compact-list.png"),
  });
  const after: Trade[] = await (
    await request.get(`${api}/trades?account_id=${accountId}`, { headers })
  ).json();
  const persistent = (rows: Trade[]) => rows.map(({ holding: _holding, ...tr }) => tr);
  expect(persistent(after)).toEqual(persistent(saved));
});
