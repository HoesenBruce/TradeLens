import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";

// Only run against a disposable API with a synthetic user.
test.use({ serviceWorkers: "block", viewport: { width: 1440, height: 1000 } });
const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";
const [title, columns, ...data] = readFileSync(
  "../api/internal/importer/testdata/sbi-review-direction.csv",
  "utf8",
)
  .trim()
  .split("\n");
const header = `${title}\n${columns}\n`;
const rows = data;

test("SBI closing-first import reconciles review roles, re-entry and duplicates", async ({
  page,
  request,
}, info) => {
  test.setTimeout(90_000);
  const login = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const accountResponse = await request.post(`${api}/accounts`, {
    headers,
    data: {
      name: "Synthetic issue 307",
      broker: "SBI Securities",
      account_type: "margin",
      base_currency: "JPY",
      starting_balance: 0,
    },
  });
  expect(accountResponse.status()).toBe(201);
  const account = await accountResponse.json();
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
    { token, api, accountId: account.id },
  );
  const fixture = {
    name: "synthetic-307.csv",
    mimeType: "text/csv",
    buffer: Buffer.from(header + rows.join("\n") + "\n"),
  };
  const trades = async () => {
    const response = await request.get(`${api}/trades?account_id=${account.id}`, { headers });
    expect(response.ok()).toBeTruthy();
    return response.json();
  };
  const preview = async () => {
    await page.goto(`${web}/import`);
    await page.getByRole("combobox", { name: "Account select" }).selectOption(account.id);
    await page.locator('input[type="file"]').setInputFiles(fixture);
    await page.getByRole("button", { name: "Preview import", exact: true }).click();
    await expect(page.getByRole("button", { name: "Confirm import", exact: true })).toBeVisible();
  };
  await preview();
  expect(await trades()).toHaveLength(0);
  await page.getByRole("button", { name: "Back", exact: true }).click();
  expect(await trades()).toHaveLength(0);
  await preview();
  await page.screenshot({ path: info.outputPath("preview.png") });
  const commit = async (inserted: number, skipped: number) => {
    const response = page.waitForResponse(
      (r) => r.url() === `${api}/imports/commit` && r.request().method() === "POST",
    );
    await page.getByRole("button", { name: "Confirm import", exact: true }).click();
    const result = await response;
    expect(result.ok(), await result.text()).toBeTruthy();
    expect(await result.json()).toMatchObject({ inserted, skipped });
    await expect(page.getByText("Import finished", { exact: true })).toBeVisible();
  };
  await commit(9, 0);
  const saved = await trades();
  expect(saved).toHaveLength(4);
  const margin = saved.find((t: { qty_opened: number }) => t.qty_opened === 20);
  const cash = saved.find((t: { qty_opened: number }) => t.qty_opened === 10);
  const short = saved.find((t: { direction: string }) => t.direction === "short");
  const missing = saved.find((t: { symbol: string }) => t.symbol === "734A");
  expect(margin).toMatchObject({
    direction: "long",
    avg_entry_price: 101,
    avg_exit_price: 111,
    qty_remaining: 0,
    net_pnl: 200,
    fees_total: 4,
    time_in_trade_secs: null,
  });
  expect(cash).toMatchObject({
    direction: "long",
    avg_entry_price: 100,
    avg_exit_price: 110,
    qty_opened: 10,
    qty_remaining: 0,
  });
  expect(short).toMatchObject({
    qty_opened: 70,
    qty_remaining: 10,
    direction: "short",
    net_pnl: null,
  });
  expect(missing).toMatchObject({
    direction: "long",
    qty_opened: 0,
    qty_remaining: 0,
    net_pnl: null,
  });
  expect(missing.accounting_warning).toContain("Incomplete SBI opening evidence");
  await page.goto(`${web}/trades`);
  await expect(page.locator("tbody tr")).toHaveCount(4);
  await expect(page.getByLabel("Open Qty 10 / Entry Qty 70", { exact: true })).toBeVisible();
  await page.screenshot({ path: info.outputPath("list.png") });
  await page.locator("tbody tr").first().click();
  await expect(page.getByRole("dialog")).toContainText("SBI date-level reconciliation");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).not.toBeVisible();
  await page.locator("tbody tr").first().click();
  await expect(page.getByRole("dialog")).toContainText("SBI date-level reconciliation");
  await page.getByRole("button", { name: "Open full page" }).click();
  await expect(page.getByText("Back to trades")).toBeVisible();
  for (const [name, trade] of [
    ["margin", margin],
    ["cash", cash],
    ["short", short],
    ["missing", missing],
  ] as const) {
    await page.goto(`${web}/trades/${trade.id}`);
    await expect(
      page.getByRole("alert").filter({ hasText: "SBI date-level reconciliation" }),
    ).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Pos", exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Replay trade" })).toHaveCount(0);
    if (name === "short")
      await expect(page.getByText("Open Qty", { exact: true }).locator("..")).toBeVisible();
    if (name === "margin") {
      await expect(page.getByText("Sep 1, 2026 → Sep 1, 2026", { exact: true })).toBeVisible();
      await page.getByRole("button", { name: "1D", exact: true }).click();
      await expect(page.locator("canvas").first()).toBeVisible();
      await page.screenshot({ path: info.outputPath("daily-chart.png") });
      await page.getByRole("button", { name: "Expand chart", exact: true }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
      await page.keyboard.press("Escape");
      await expect(page.getByRole("dialog")).not.toBeVisible();
      await page.getByRole("button", { name: "1m", exact: true }).click();
      await expect(page.locator("canvas").first()).toBeVisible();
    }
    await page.screenshot({ path: info.outputPath(`${name}-detail.png`), fullPage: true });
    const executions = page.getByRole("heading", {
      name: `Executions (${name === "margin" ? 4 : name === "missing" ? 1 : 2})`,
      exact: true,
    });
    await executions.scrollIntoViewIfNeeded();
    await expect(executions).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Time", exact: true })).toHaveCount(0);
    await expect(page.getByRole("columnheader", { name: "Executed", exact: true })).toBeVisible();
    await page.screenshot({ path: info.outputPath(`${name}-executions.png`) });
    await page.reload();
    await expect(
      page.getByRole("alert").filter({ hasText: "SBI date-level reconciliation" }),
    ).toBeVisible();
  }
  await preview();
  await commit(0, 9);
  expect(await trades()).toEqual(saved);
  // An alternate file order must preserve determinable aggregates and identities.
  fixture.buffer = Buffer.from(header + [...rows].reverse().join("\n") + "\n");
  await preview();
  await commit(0, 9);
  expect(await trades()).toEqual(saved);
});
