import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import type { Trade, TradeDetail } from "../src/lib/api/types";

// Disposable API only: synthetic accounts/fills; no production history repair.
test.use({
  actionTimeout: 10_000,
  serviceWorkers: "block",
  viewport: { width: 1440, height: 1000 },
});
const api = process.env.E2E_API_URL ?? "http://localhost:8092/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5174";

test("quantities agree across broker paths, sorting, details, locales and compact Web", async ({
  page,
  request,
}, info) => {
  test.setTimeout(180_000);
  const credentials = {
    email: process.env.E2E_EMAIL ?? "issue316@synthetic.invalid",
    password: process.env.E2E_PASSWORD ?? "synthetic-316-password",
  };
  await request.post(`${api}/setup`, { data: { ...credentials, name: "Synthetic quantity QA" } });
  const login = await request.post(`${api}/auth/login`, { data: credentials });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const account = async (broker: string) => {
    const response = await request.post(`${api}/accounts`, {
      headers,
      data: {
        name: `Synthetic 308 ${broker}`,
        broker,
        account_type: "margin",
        base_currency: "JPY",
        starting_balance: 0,
      },
    });
    expect(response.status()).toBe(201);
    return response.json();
  };
  const manual = await account("Manual"),
    ibkr = await account("IBKR"),
    generic = await account("Generic CSV"),
    sbi = await account("SBI Securities"),
    edge = await account("Manual"),
    virtual = await account("Manual");
  const fills = async (accountId: string, symbol: string, sides: string[], qty: number[]) => {
    for (let i = 0; i < sides.length; i++) {
      const response = await request.post(`${api}/executions`, {
        headers,
        data: {
          account_id: accountId,
          symbol,
          instrument_type: "stock",
          side: sides[i],
          quantity: qty[i],
          price: sides[i] === "buy" ? 100 : 110,
          fees: 1,
          executed_at: `2026-09-01T0${i + 1}:00:00Z`,
        },
      });
      expect(response.status(), await response.text()).toBe(201);
    }
  };
  await fills(manual.id, "PART", ["buy", "sell"], [500, 400]);
  await fills(manual.id, "PYR", ["buy", "sell", "buy"], [500, 400, 200]);
  await fills(manual.id, "FULL", ["buy", "sell"], [500, 500]);
  await fills(manual.id, "SHORT", ["sell", "buy"], [70, 60]);
  await fills(manual.id, "REV", ["buy", "sell"], [500, 700]);
  await fills(edge.id, "E2ELONG", ["buy", "sell"], [1234567890123.125, 1234567890000]);
  await fills(edge.id, "E2EFRAC", ["buy", "sell"], [0.123456789, 0.023456789]);
  await fills(edge.id, "E2EOPEN", ["buy"], [1250]);
  await fills(ibkr.id, "PART", ["buy", "sell"], [500, 400]);
  const importCSV = async (accountId: string, buffer: Buffer, mapping?: Record<string, string>) => {
    const file = { name: "synthetic-308.csv", mimeType: "text/csv", buffer };
    if (!mapping) {
      const preview = await request.post(`${api}/imports`, {
        headers,
        multipart: { account_id: accountId, file },
      });
      expect(preview.ok()).toBeTruthy();
      mapping = (await preview.json()).suggested_mapping;
    }
    const response = await request.post(`${api}/imports/commit`, {
      headers,
      multipart: {
        account_id: accountId,
        file,
        column_mapping: JSON.stringify(mapping),
      },
    });
    expect(response.ok(), await response.text()).toBeTruthy();
  };
  await importCSV(
    generic.id,
    Buffer.from(
      "Symbol,Side,Qty,Price,Time\nPART,Buy,500,100,2026-09-01T01:00:00Z\nPART,Sell,400,110,2026-09-01T02:00:00Z\n",
    ),
    { symbol: "Symbol", side: "Side", quantity: "Qty", price: "Price", executed_at: "Time" },
  );
  await importCSV(
    sbi.id,
    readFileSync("../api/internal/importer/testdata/sbi-review-direction.csv"),
  );
  const sbiHeader =
    "約定履歴照会\n約定日,銘柄,銘柄コード,市場,取引,約定数量,約定単価,手数料/諸経費等,受渡金額/決済損益\n";
  await importCSV(
    sbi.id,
    Buffer.from(
      sbiHeader +
        "2026/09/01,Synthetic CONVERSION,735A,東証,信用新規買,100,1000,--,--\n2026/09/02,Synthetic CONVERSION,735A,東証,現引,50,1000,--,--\n",
    ),
  );
  const virtualRows = Array.from({ length: 30 }, (_, index) => {
    const day = String(index + 1).padStart(2, "0");
    return `E2EV${day},Buy,${(index + 1) * 100},100,2026-07-${day}T01:00:00Z\nE2EV${day},Sell,${(index + 1) * 50},110,2026-07-${day}T02:00:00Z`;
  });
  await importCSV(
    virtual.id,
    Buffer.from("Symbol,Side,Qty,Price,Time\n" + virtualRows.join("\n") + "\n"),
    { symbol: "Symbol", side: "Side", quantity: "Qty", price: "Price", executed_at: "Time" },
  );
  const trades = async (accountId: string): Promise<Trade[]> => {
    const response = await request.get(`${api}/trades?account_id=${accountId}`, { headers });
    expect(response.ok()).toBeTruthy();
    return response.json();
  };
  const saved = await trades(manual.id);
  for (const accountId of [manual.id, ibkr.id, generic.id]) {
    const partial = (await trades(accountId)).find((t) => t.symbol === "PART")!;
    expect(partial.quantities).toMatchObject({
      entry_qty: 500,
      exit_qty: 400,
      open_qty: 100,
      remaining_pct: 20,
      lifecycle: "partially_closed",
    });
  }
  const sbiSaved = await trades(sbi.id);
  expect(sbiSaved.find((t) => t.symbol === "734A")?.quantities).toMatchObject({
    entry_qty: null,
    exit_qty: null,
    open_qty: null,
    remaining_pct: null,
    lifecycle: "unknown",
  });
  expect(sbiSaved.filter((t) => t.symbol === "735A").map((t) => t.quantities?.open_qty)).toEqual([
    50, 50,
  ]);
  const edgeSaved = await trades(edge.id);
  const all = [...saved, ...sbiSaved, ...edgeSaved];
  for (const row of all) {
    const response = await request.get(`${api}/trades/${row.id}`, { headers });
    expect(response.ok()).toBeTruthy();
    const detail: TradeDetail = await response.json();
    expect(detail.quantities).toEqual(row.quantities);
    expect(detail.net_pnl).toBe(row.net_pnl);
    expect(detail.fees_total).toBe(row.fees_total);
  }
  await page.addInitScript(
    ({ token, api, accountId }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      if (!localStorage.getItem("tm-locale")) localStorage.setItem("tm-locale", "en");
      if (!localStorage.getItem("tm_filters"))
        localStorage.setItem(
          "tm_filters",
          JSON.stringify({ state: { accountIds: [accountId] }, version: 1 }),
        );
    },
    { token, api, accountId: manual.id },
  );
  await page.goto(`${web}/trades`);
  const quantityHeader = page.getByRole("button", { name: "Close progress", exact: true });
  await expect(page.locator('[role="progressbar"][aria-valuenow="80"]')).toBeVisible();
  await page.waitForTimeout(350);
  await page.screenshot({ animations: "disabled", path: info.outputPath("list.png") });
  const order = async () =>
    page.locator('tbody [data-slot="trade-quantity"]').evaluateAll((cells) =>
      cells.map(
        (cell) =>
          cell
            .querySelector('[role="progressbar"]')
            ?.getAttribute("aria-valuetext")
            ?.match(/Remaining ([^ ]+) /)?.[1] ?? "unknown",
      ),
    );
  await quantityHeader.click();
  await page.getByRole("menuitem", { name: "Asc", exact: true }).click();
  expect(await order()).toEqual(["0", "0", "10", "100", "200", "300"]);
  await quantityHeader.click();
  await page.getByRole("menuitem", { name: "Desc", exact: true }).click();
  expect(await order()).toEqual(["300", "200", "100", "10", "0", "0"]);
  await quantityHeader.click();
  await page.getByRole("menu").focus();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).not.toBeVisible();
  await page.getByRole("button", { name: "Add filter", exact: true }).click();
  await page.getByPlaceholder("Search filters…").fill("Symbol");
  await page.getByPlaceholder("Search filters…").press("ArrowDown");
  await page.getByPlaceholder("Search filters…").press("ArrowRight");
  const symbolSearch = page.getByPlaceholder("Search symbol...");
  await expect(symbolSearch).toBeFocused();
  await symbolSearch.fill("PART");
  await symbolSearch.press("ArrowDown");
  await symbolSearch.press("Enter");
  await page.keyboard.press("Escape");
  await expect(page.locator("tbody tr")).toHaveCount(1);
  const symbolFilter = page
    .locator('[data-slot="button-group"]')
    .filter({ has: page.getByText("Symbol", { exact: true }) });
  await symbolFilter.getByRole("button").last().click();
  await expect(page.locator("tbody tr")).toHaveCount(6);
  expect((await order())[0]).toBe("300");
  await quantityHeader.click();
  await page.getByRole("menuitem", { name: "Reset", exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menu")).not.toBeVisible();
  const row = page
    .locator("tbody tr")
    .filter({ has: page.locator('[role="progressbar"][aria-valuenow="80"]') });
  await row.click();
  await expect(
    page.getByRole("dialog").getByText("Partially closed", { exact: true }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).not.toBeVisible();
  await row.click();
  await expect(
    page.getByRole("dialog").getByText("Open Qty", { exact: true }).locator(".."),
  ).toContainText("100");
  await page.getByRole("button", { name: "Open full page", exact: true }).click();
  const partial = saved.find((t) => t.symbol === "PART")!;
  for (const [label, value] of [
    ["Entry Qty", "500"],
    ["Exit Qty", "400"],
    ["Open Qty", "100"],
    ["Remaining %", "20.0%"],
  ])
    await expect(page.getByText(label, { exact: true }).locator("..")).toContainText(value);
  await page.waitForTimeout(350);
  await page.screenshot({ animations: "disabled", path: info.outputPath("partial-detail.png") });
  await page.reload();
  await expect(page.getByText("Open Qty", { exact: true }).locator("..")).toContainText("100");
  for (const [locale, label] of [
    ["zh-CN", "剩余量"],
    ["ja", "残数量"],
    ["en", "Open Qty"],
  ]) {
    await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
    await page.reload();
    await expect(page.getByText(label, { exact: true }).locator("..")).toContainText("100");
  }
  for (const symbol of ["SHORT", "FULL"]) {
    const tr = saved.find((t) => t.symbol === symbol)!;
    await page.goto(`${web}/trades/${tr.id}`);
    await expect(page.getByText("Open Qty", { exact: true }).locator("..")).toContainText(
      symbol === "SHORT" ? "10" : "0",
    );
    await page.waitForTimeout(350);
    await page.screenshot({
      animations: "disabled",
      path: info.outputPath(`${symbol.toLowerCase()}-detail.png`),
    });
  }
  const unknown = sbiSaved.find((t) => t.symbol === "734A")!;
  await page.goto(`${web}/trades/${unknown.id}`);
  await expect(
    page.getByText("Quantity is unknown because opening history is incomplete."),
  ).toBeVisible();
  await expect(page.getByText("Open Qty", { exact: true }).locator("..")).toContainText("—");
  await page.waitForTimeout(350);
  await page.screenshot({ animations: "disabled", path: info.outputPath("unknown.png") });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${web}/trades`);
  await expect(page.locator('[role="progressbar"][aria-valuenow="80"]')).toBeVisible();
  await page.waitForTimeout(350);
  await page.screenshot({ animations: "disabled", path: info.outputPath("compact-list.png") });
  await page.goto(`${web}/trades/${partial.id}`);
  await expect(page.getByText("Open Qty", { exact: true }).locator("..")).toContainText("100");
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
  ).toBeTruthy();
  await page.waitForTimeout(350);
  await page.screenshot({ animations: "disabled", path: info.outputPath("compact-detail.png") });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.evaluate(
    (id) =>
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [id] }, version: 1 }),
      ),
    sbi.id,
  );
  await page.goto(`${web}/trades`);
  await expect(
    page.locator('tbody [data-slot="trade-quantity"]').filter({ hasText: "Position unknown" }),
  ).toBeVisible();
  for (const direction of ["Asc", "Desc"]) {
    await page.getByRole("button", { name: "Close progress", exact: true }).click();
    await page.getByRole("menuitem", { name: direction, exact: true }).click();
    expect((await order()).at(-1)).toBe("unknown");
  }
  await page.screenshot({ animations: "disabled", path: info.outputPath("unknown-list.png") });
  await page.evaluate(
    (id) =>
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [id] }, version: 1 }),
      ),
    edge.id,
  );
  await page.goto(`${web}/trades`);
  await expect(
    page.getByText("Closed 1,234,567,889,999.995 / 1,234,567,890,123.125", { exact: false }),
  ).toBeVisible();
  await expect(page.getByText("Remaining 0.1 (81.0%)", { exact: true })).toBeVisible();
  await page.screenshot({
    animations: "disabled",
    path: info.outputPath("long-fraction-list.png"),
  });
  const long = edgeSaved.find((t) => t.symbol === "E2ELONG")!;
  // Keep the existing aggregated inventory's two-decimal remaining-quantity policy.
  expect(long.quantities?.open_qty).toBe(123.13);
  await expect(page.getByText("Remaining 123.13 (<0.1%)", { exact: true })).toBeVisible();
  const readable = async () => {
    const values = page.locator('[data-slot="trade-quantity"] [data-slot="quantity-value"]');
    await expect(values).not.toHaveCount(0);
    for (const value of await values.all()) {
      await expect(value).toBeVisible();
      expect(await value.evaluate((el) => el.scrollWidth <= el.clientWidth + 1)).toBeTruthy();
    }
  };
  await readable();
  await page
    .locator("tbody tr")
    .filter({ has: page.getByText("E2ELONG", { exact: true }) })
    .click();
  await expect(
    page.getByRole("dialog").getByText("1,234,567,890,123.125", { exact: true }),
  ).toBeVisible();
  await readable();
  await page.screenshot({ animations: "disabled", path: info.outputPath("long-peek.png") });
  await page.keyboard.press("Escape");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.reload();
  await readable();
  await page.screenshot({ animations: "disabled", path: info.outputPath("long-compact-list.png") });
  const longCard = page
    .locator('[tabindex="0"]')
    .filter({ has: page.getByText("E2ELONG", { exact: true }) })
    .first();
  await longCard.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("dialog").getByText("1,234,567,890,123.125", { exact: true }),
  ).toBeVisible();
  await readable();
  await page.screenshot({ animations: "disabled", path: info.outputPath("long-compact-peek.png") });
  await page.getByRole("button", { name: "Open full page", exact: true }).click();
  for (const locale of ["en", "zh-CN", "ja"]) {
    await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
    await page.reload();
    await expect(page.getByText("1,234,567,890,123.125", { exact: true })).toBeVisible();
    await readable();
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    ).toBeTruthy();
    await page.screenshot({
      animations: "disabled",
      path: info.outputPath(`long-compact-detail-${locale}.png`),
    });
  }
  expect(page.url()).toContain(long.id);
  const stable = (rows: Trade[]) =>
    rows.map((row) => ({
      ...row,
      holding: row.holding?.ongoing ? { ...row.holding, seconds: null } : row.holding,
    }));
  expect(stable(await trades(manual.id))).toEqual(stable(saved));
  expect(stable(await trades(edge.id))).toEqual(stable(edgeSaved));
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.evaluate((id) => {
    localStorage.setItem("tm-locale", "en");
    localStorage.setItem("tm_filters", JSON.stringify({ state: { accountIds: [id] }, version: 1 }));
  }, virtual.id);
  await page.goto(`${web}/trades`);
  await page.getByRole("button", { name: "Close progress", exact: true }).click();
  await page.getByRole("menuitem", { name: "Asc", exact: true }).click();
  await page.getByRole("combobox", { name: "Rows per page", exact: true }).selectOption("50");
  await expect(page.getByText("E2EV01", { exact: true })).toBeVisible();
  const scroller = page.locator("table").locator("..");
  await scroller.evaluate((el) => {
    el.scrollTop = el.scrollHeight;
  });
  await expect(page.getByText("E2EV30", { exact: true })).toBeVisible();
  await scroller.evaluate((el) => {
    el.scrollTop = 0;
  });
  await expect(page.getByText("E2EV01", { exact: true })).toBeVisible();
  await page.screenshot({
    animations: "disabled",
    path: info.outputPath("virtualized-quantity-rows.png"),
  });
});
