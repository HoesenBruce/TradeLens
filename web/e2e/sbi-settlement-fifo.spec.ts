import { expect, test } from "@playwright/test";
const api = process.env.E2E_API_URL ?? "http://127.0.0.1:8097/api/v1";
const web = process.env.E2E_BASE_URL ?? "http://127.0.0.1:5197";
test.use({ serviceWorkers: "block", viewport: { width: 1440, height: 1000 } });

// Disposable API and generated synthetic records only.
test("SBI FIFO preview, cancel, commit, read-back, re-entry, replay and atomic refusal", async ({
  page,
  request,
}, info) => {
  test.setTimeout(180_000);
  const login = await request.post(`${api}/auth/login`, {
    data: { email: "synthetic347@example.invalid", password: "Synthetic347-test-pass" },
  });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const response = await request.post(`${api}/accounts`, {
    headers,
    data: { name: `Synthetic #347 ${Date.now()}`, broker: "sbi", base_currency: "JPY" },
  });
  expect(response.ok()).toBeTruthy();
  const account = await response.json();
  const history =
    "約定履歴照会\n約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等,信用区分,金利,貸株料,その他諸経費,受渡金額/決済損益\n" +
    "2032-03-02T09:00:00+09:00,9101,信用新規買,100,100,0,制度信用,0,0,0,--\n2032-03-02T10:00:00+09:00,9101,信用新規買,100,100,0,制度信用,0,0,0,--\n2032-03-02T11:00:00+09:00,9101,信用返済売,150,110,0,制度信用,0,0,0,1500\n";
  const file = (body: string) => ({
    name: "synthetic-347.csv",
    mimeType: "text/csv",
    buffer: Buffer.from(body),
  });
  const seeded = await request.post(`${api}/imports/commit`, {
    headers,
    multipart: {
      account_id: account.id,
      file: file(history),
      column_mapping: JSON.stringify({
        symbol: "銘柄コード",
        side: "取引",
        quantity: "約定数量",
        price: "約定単価",
        executed_at: "約定日",
        fees: "手数料/諸経費等",
        instrument_type: "=stock",
      }),
    },
  });
  expect(seeded.ok(), await seeded.text()).toBeTruthy();
  await page.addInitScript(
    ({ token, api }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
    },
    { token, api },
  );
  const report =
    "実現損益詳細\n約定日,銘柄コード,取引,数量,単価,平均取得価額,実現損益\n2032/03/02,9101,信用返済売,150,110,100,1500\n";
  const executions = async () =>
    (await request.get(`${api}/executions?account_id=${account.id}`, { headers })).json();
  const before = await executions();
  const preview = async (body: string) => {
    await page.goto(`${web}/import`);
    await page.getByRole("combobox", { name: "Account select" }).selectOption(account.id);
    await page.locator('input[type="file"]').setInputFiles(file(body));
    await page.getByRole("button", { name: "Preview import", exact: true }).click();
    await expect(page.getByRole("button", { name: "Confirm import", exact: true })).toBeVisible();
  };
  const confirm = async () => {
    const result = page.waitForResponse(
      (r) => r.url() === `${api}/imports/commit` && r.request().method() === "POST",
    );
    await page.getByRole("button", { name: "Confirm import", exact: true }).click();
    return result;
  };
  await preview(report);
  await expect(page.getByText(/fifo_inferred: TradeLens/)).toBeVisible({ timeout: 30_000 });
  await page.screenshot({ path: info.outputPath("fifo-preview.png"), fullPage: true });
  expect(await executions()).toEqual(before);
  await page.getByRole("button", { name: "Back", exact: true }).click();
  expect(await executions()).toEqual(before);
  await preview(report);
  const committed = await confirm();
  expect(committed.ok(), await committed.text()).toBeTruthy();
  expect(await committed.json()).toMatchObject({
    enriched: 1,
    enrichment_rows: [{ match_provenance: "exact", opening_match_provenance: "fifo_inferred" }],
  });
  await expect(page.getByText("Import finished", { exact: true })).toBeVisible();
  const after = await executions();
  expect(after).toHaveLength(3);
  expect(
    after.find((e: { side: string }) => e.side === "sell").details.sbi_opening_match_provenance,
  ).toBe("fifo_inferred");
  const trades = await (
    await request.get(`${api}/trades?account_id=${account.id}`, { headers })
  ).json();
  expect(trades).toHaveLength(1);
  expect(trades[0]).toMatchObject({ qty_opened: 200, qty_remaining: 50, time_in_trade_secs: null });
  await page.goto(`${web}/trades/${trades[0].id}`);
  await expect(page.getByText(/fifo_inferred: TradeLens/)).toBeVisible({ timeout: 30_000 });
  await page.screenshot({ path: info.outputPath("fifo-detail.png"), fullPage: true });
  await page.reload();
  await expect(page.getByText(/fifo_inferred: TradeLens/)).toBeVisible({ timeout: 30_000 });
  await preview(report);
  await expect(page.getByText(/Already enriched/).first()).toBeVisible();
  expect((await confirm()).ok()).toBeTruthy();
  await expect(page.getByText("Import finished", { exact: true })).toBeVisible();
  expect(await executions()).toEqual(after);
  await preview(report.replace(",100,1500", ",100,1499"));
  await expect(page.getByText(/Conflict/).first()).toBeVisible();
  expect((await confirm()).status()).toBe(409);
  expect(await executions()).toEqual(after);
  await page.screenshot({ path: info.outputPath("conflict-refused.png"), fullPage: true });
});
