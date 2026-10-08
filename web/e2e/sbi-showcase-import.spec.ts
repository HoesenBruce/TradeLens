/// <reference types="node" />
import { execFileSync } from "node:child_process";
import { expect, test } from "@playwright/test";

// Run only against a disposable API; creates a separate synthetic account per run.
test.use({ serviceWorkers: "block", viewport: { width: 1440, height: 1000 } });
const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";

test("SBI synthetic CSV previews without writes, commits and deduplicates", async ({
  page,
  request,
}, info) => {
  const login = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const created = await request.post(`${api}/accounts`, {
    headers,
    data: {
      name: "[FICTIONAL showcase v1] SBI upload QA",
      broker: "SBI Securities",
      account_type: "margin",
      base_currency: "JPY",
      starting_balance: 0,
    },
  });
  expect(created.status()).toBe(201);
  const account = await created.json();
  const buffer = execFileSync("python3", [
    "-c",
    "import importlib.util; s=importlib.util.spec_from_file_location('showcase','../scripts/seed-showcase.py'); m=importlib.util.module_from_spec(s); s.loader.exec_module(m); print(m.trade_csv(),end='')",
  ]);
  const fixture = { name: "fictional-showcase-v1.csv", mimeType: "text/csv", buffer };
  const trades = async () => {
    const response = await request.get(`${api}/trades?account_id=${account.id}`, { headers });
    expect(response.ok()).toBeTruthy();
    return response.json();
  };
  const executions = async () => {
    const response = await request.get(`${api}/executions?account_id=${account.id}`, { headers });
    expect(response.ok()).toBeTruthy();
    return response.json();
  };
  await page.addInitScript(
    ({ token, api }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
    },
    { token, api },
  );
  await page.goto(`${web}/import`);
  await page.getByRole("combobox", { name: "Account select" }).selectOption(account.id);
  const upload = page.locator('input[type="file"]');
  await upload.setInputFiles(fixture);
  await expect(page.getByRole("button", { name: "Preview import", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "Remove file", exact: true }).click();
  await expect(page.getByRole("button", { name: "Preview import", exact: true })).toBeDisabled();
  const preview = async () => {
    await page.getByRole("combobox", { name: "Account select" }).selectOption(account.id);
    await upload.setInputFiles(fixture);
    const response = page.waitForResponse(
      (r) => r.url() === `${api}/imports` && r.request().method() === "POST",
    );
    await page.getByRole("button", { name: "Preview import", exact: true }).click();
    const result = await response;
    expect(result.ok()).toBeTruthy();
    expect((await result.json()).detected_broker).toBe("SBI Securities (Execution History)");
    await expect(page.getByRole("button", { name: "Confirm import", exact: true })).toBeVisible();
  };
  await preview();
  expect(await trades()).toHaveLength(0);
  expect(await executions()).toHaveLength(0);
  await page.getByRole("button", { name: "Back", exact: true }).click();
  expect(await trades()).toHaveLength(0);
  expect(await executions()).toHaveLength(0);
  await preview();
  await page.screenshot({ path: info.outputPath("sbi-preview.png"), fullPage: false });
  await page.getByRole("button", { name: "Confirm import", exact: true }).scrollIntoViewIfNeeded();
  await page.screenshot({ path: info.outputPath("sbi-confirm.png"), fullPage: false });
  const commit = async (inserted: number, skipped: number) => {
    const response = page.waitForResponse(
      (r) => r.url() === `${api}/imports/commit` && r.request().method() === "POST",
    );
    await page.getByRole("button", { name: "Confirm import", exact: true }).click();
    const result = await response;
    expect(result.ok()).toBeTruthy();
    const body = await result.json();
    expect(body.inserted).toBe(inserted);
    expect(body.skipped).toBe(skipped);
    expect(body.errors ?? []).toHaveLength(0);
    await expect(page.getByText("Import finished", { exact: true })).toBeVisible();
    await expect(page.getByText("Inserted", { exact: true }).locator("..")).toContainText(
      String(inserted),
    );
    await expect(
      page.getByText("Skipped (duplicates)", { exact: true }).locator(".."),
    ).toContainText(String(skipped));
  };
  await commit(14, 0);
  const savedExecutions = await executions();
  expect(savedExecutions).toHaveLength(14);
  const saved = await trades();
  expect(saved).toHaveLength(7);
  expect(saved.reduce((sum: number, t: { net_pnl: number }) => sum + t.net_pnl, 0)).toBe(25000);
  expect(saved.every((t: { qty_remaining: number }) => t.qty_remaining === 0)).toBeTruthy();
  expect(new Set(saved.map((t: { symbol: string }) => t.symbol))).toEqual(
    new Set(["6501", "285A", "7203", "6758", "8306"]),
  );
  await page.screenshot({ path: info.outputPath("sbi-result.png"), fullPage: false });
  await page.reload();
  await page.getByRole("combobox", { name: "Account select" }).selectOption(account.id);
  await preview();
  await commit(0, 14);
  expect(await trades()).toEqual(saved);
  expect(await executions()).toEqual(savedExecutions);
  await page.screenshot({ path: info.outputPath("sbi-duplicates.png"), fullPage: false });
  await page.getByRole("button", { name: "Import another", exact: true }).click();
  await expect(page.getByRole("button", { name: "Preview import", exact: true })).toBeDisabled();
});
