import { expect, test } from "@playwright/test";
// Each run uses a fresh browser context and verifies persisted fields through the live API.
test.use({ serviceWorkers: "allow" });
const api = process.env.E2E_API_URL ?? "http://localhost:8091/api/v1";
test("review queue preserves fields, navigation, account scope, errors and dismissal", async ({
  page,
  request,
}, info) => {
  const setup = await request.post(`${api}/setup`, {
    data: { email: "review273@example.com", password: "Review273-test-pass" },
  });
  const auth = setup.ok()
    ? await setup.json()
    : await (
        await request.post(`${api}/auth/login`, {
          data: { email: "review273@example.com", password: "Review273-test-pass" },
        })
      ).json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  async function post(path: string, data: unknown) {
    const r = await request.post(`${api}${path}`, { headers, data });
    expect(r.ok(), await r.text()).toBeTruthy();
    return r.json();
  }
  const a = await post("/accounts", {
    name: `Review A ${Date.now()}`,
    broker: "manual",
    account_type: "cash",
    base_currency: "USD",
  });
  const b = await post("/accounts", {
    name: `Review B ${Date.now()}`,
    broker: "manual",
    account_type: "cash",
    base_currency: "USD",
  });
  const custom = await post("/tags", { name: `Custom ${Date.now()}`, kind: "custom" });
  const mistake = await post("/tags", { name: `Mistake ${Date.now()}`, kind: "mistake" });
  const setups = await Promise.all(
    ["One", "Two"].map((name) => post("/setups", { name: `${name} ${Date.now()}` })),
  );
  async function trade(account: string, symbol: string, days: number) {
    const close = Date.now() - days * 86400000;
    for (const [side, offset, price] of [
      ["buy", -3600000, 100],
      ["sell", 0, 110],
    ] as const)
      await post("/executions", {
        account_id: account,
        symbol,
        instrument_type: "stock",
        side,
        quantity: 1,
        price,
        executed_at: new Date(close + offset).toISOString(),
      });
    const list = await request.get(`${api}/trades?account_id=${account}&target_currency=USD`, {
      headers,
    });
    expect(list.ok(), await list.text()).toBeTruthy();
    const rows = await list.json();
    return rows.trades.find((t: { symbol: string }) => t.symbol === symbol);
  }
  const first = await trade(a.id, "REVIEW273A", 2),
    second = await trade(a.id, "REVIEW273B", 1),
    old = await trade(a.id, "OLD273A", 20),
    other = await trade(b.id, "OLD273B", 21);
  const notes =
    "Legacy preserved\n\n## Session\nAsia\n\n## Planned direction\nshort\n\n## Entry reason\nentry\n\n## Exit reason\nexit\n\n## Review notes\nold lesson";
  expect(
    (
      await request.patch(`${api}/trades/${first.id}`, {
        headers,
        data: {
          notes,
          setup_ids: setups.map((s) => s.id),
          initial_risk: 20,
          target_price: 150,
          stop_price: 80,
          confidence: 4,
          tag_ids: [custom.id, mistake.id],
        },
      })
    ).ok(),
  ).toBeTruthy();
  const before = await (await request.get(`${api}/trades/${first.id}`, { headers })).json();
  await page.addInitScript(
    ({ token, api, ids }) => {
      if (localStorage.getItem("review273-seeded")) return;
      localStorage.setItem("review273-seeded", "true");
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({
          state: { accountIds: ids, from: "2000-01-01", to: "2000-01-02", symbols: ["NO_MATCH"] },
          version: 1,
        }),
      );
    },
    { token: auth.access_token, api, ids: [a.id] },
  );
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/review");
  await expect(page.getByRole("button", { name: "Recent (2)" })).toBeVisible();
  await expect(page.getByRole("heading", { name: /REVIEW273A/ })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Review notes" })).toHaveValue("old lesson");
  await expect(page.getByRole("checkbox", { name: custom.name })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Save and next" })).toBeDisabled();
  await page.getByRole("textbox", { name: "Review notes" }).fill("abandoned");
  await page.getByRole("button", { name: "A+", exact: true }).click();
  await page.getByRole("button", { name: "Cancel edits" }).click();
  await expect(page.getByRole("textbox", { name: "Review notes" })).toHaveValue("old lesson");
  await page.getByRole("button", { name: "Skip", exact: true }).click();
  await expect(page.getByRole("heading", { name: /REVIEW273B/ })).toBeVisible();
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "Review notes" })).toHaveValue("old lesson");
  await page.getByRole("button", { name: "A+", exact: true }).click();
  await page.getByRole("textbox", { name: "Review notes" }).fill("new lesson");
  await page.getByRole("checkbox", { name: mistake.name }).uncheck();
  let failed = false;
  let writes = 0;
  await page.route(`**/trades/${first.id}`, async (route) => {
    if (route.request().method() === "PATCH") writes++;
    if (route.request().method() === "PATCH" && !failed) {
      failed = true;
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: { code: "internal", message: "test failure" } }),
      });
    } else await route.continue();
  });
  await page.getByRole("button", { name: "Save and next" }).click();
  await expect(
    page.getByText("Save failed. Your draft is retained; retry or cancel."),
  ).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Review notes" })).toHaveValue("new lesson");
  await page.screenshot({ path: info.outputPath("review-error.png"), fullPage: true });
  await page.getByRole("button", { name: "Save and next" }).click();
  await expect(page.getByRole("heading", { name: /REVIEW273B/ })).toBeVisible();
  const after = await (await request.get(`${api}/trades/${first.id}`, { headers })).json();
  for (const key of Object.keys(before).filter(
    (k) => !["notes", "tags", "trade_quality"].includes(k),
  ))
    expect(after[key], key).toEqual(before[key]);
  expect(after.trade_quality).toBe(5);
  expect(writes).toBe(2); // One failed request and one successful retry.
  expect(after.notes).toBe(notes.replace("old lesson", "new lesson"));
  expect(after.notes).toContain("Legacy preserved");
  expect(after.notes).toContain("## Planned direction\nshort");
  expect(after.tags.map((t: { id: string }) => t.id)).toEqual([custom.id]);
  await page.reload();
  await expect(page.getByRole("button", { name: "Recent (1)" })).toBeVisible();
  await page.goto("/settings");
  await page.goto("/review");
  await expect(page.getByRole("heading", { name: /REVIEW273B/ })).toBeVisible();
  await page.getByRole("button", { name: "Skip", exact: true }).click();
  await expect(page.getByText(/^Queue complete/)).toBeVisible();
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await expect(page.getByRole("heading", { name: /REVIEW273B/ })).toBeVisible();
  await page.getByRole("button", { name: "Backlog (1)" }).click();
  await expect(page.getByRole("heading", { name: /OLD273A/ })).toBeVisible();
  await page.getByRole("button", { name: "Dismiss backlog", exact: true }).click();
  await page.getByRole("button", { name: "Cancel dismissal" }).click();
  await expect(page.getByRole("button", { name: "Backlog (1)" })).toBeVisible();
  await page.getByRole("button", { name: "Dismiss backlog", exact: true }).click();
  await page.getByRole("button", { name: "Confirm dismissal" }).click();
  await expect(page.getByRole("button", { name: "Backlog (0)" })).toBeVisible();
  expect(
    (await (await request.get(`${api}/trades/${old.id}`, { headers })).json()).trade_quality,
  ).toBeNull();
  const prefs = await (await request.get(`${api}/me/preferences`, { headers })).json();
  expect(prefs.prefs[`reviewBacklogCutoff:${a.id}`]).toBeTruthy();
  expect(prefs.prefs[`reviewBacklogCutoff:${b.id}`]).toBeUndefined();
  await page.reload();
  await expect(page.getByRole("button", { name: "Backlog (0)" })).toBeVisible();
  // A different device reads the authenticated cutoff, with no local state.
  const device = await page.context().browser()!.newContext({ serviceWorkers: "block" });
  const devicePage = await device.newPage();
  devicePage.on("pageerror", (error) => errors.push(error.message));
  await devicePage.addInitScript(
    ({ token, api, id }) => {
      // Exercise a browser without offline caching so failures reach the UI.
      Reflect.deleteProperty(Navigator.prototype, "serviceWorker");
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [id] }, version: 1 }),
      );
    },
    { token: auth.access_token, api, id: a.id },
  );
  let releaseLoad!: () => void;
  const loading = new Promise<void>((resolve) => {
    releaseLoad = resolve;
  });
  await devicePage.route("**/trades?**", async (route) => {
    await loading;
    await route.fulfill({ status: 500, body: "{}" });
  });
  await devicePage.goto(`${info.project.use.baseURL}/review`);
  await expect(devicePage.getByText("Loading…", { exact: true })).toBeVisible();
  await devicePage.screenshot({ path: info.outputPath("review-loading.png"), fullPage: true });
  releaseLoad();
  await expect(devicePage.getByText("Could not load trades.")).toBeVisible();
  await devicePage.screenshot({ path: info.outputPath("review-load-error.png"), fullPage: true });
  await devicePage.unroute("**/trades?**");
  await devicePage.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(devicePage.getByRole("button", { name: "Backlog (0)" })).toBeVisible();
  await device.close();
  await page.getByRole("button", { name: "Restore backlog" }).click();
  await expect(page.getByRole("button", { name: "Backlog (1)" })).toBeVisible();
  await page.getByRole("spinbutton", { name: "Recent days" }).fill("30");
  await expect(page.getByRole("button", { name: "Recent (2)" })).toBeVisible();
  await page.getByRole("button", { name: "Reset", exact: true }).click();
  await expect(page.getByRole("spinbutton", { name: "Recent days" })).toHaveValue("14");
  await page.screenshot({ path: info.outputPath("review-recent.png"), fullPage: true });
  expect(
    (await (await request.get(`${api}/trades/${second.id}`, { headers })).json()).trade_quality,
  ).toBeNull();
  expect(
    (await (await request.get(`${api}/trades/${other.id}`, { headers })).json()).trade_quality,
  ).toBeNull();
  await page.evaluate(
    (ids) =>
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: ids }, version: 1 }),
      ),
    [a.id, b.id],
  );
  await page.reload();
  await expect(page.getByRole("button", { name: "Recent (1)" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Backlog (2)" })).toBeVisible();
  await page.getByRole("button", { name: "Backlog (2)" }).click();
  await expect(page.getByRole("heading", { name: /OLD273B/ })).toBeVisible();
  await page.evaluate(() => localStorage.setItem("tm-locale", "zh-CN"));
  await page.reload();
  await expect(page.getByRole("heading", { name: "复盘收件箱" })).toBeVisible();
  await expect(page.getByRole("button", { name: "近期 (1)" })).toBeVisible();
  await page.screenshot({ path: info.outputPath("review-zh.png"), fullPage: true });
  await page.evaluate(() => localStorage.setItem("tm-locale", "ja"));
  await page.reload();
  await expect(page.getByRole("heading", { name: "レビュー受信箱" })).toBeVisible();
  await expect(page.getByRole("button", { name: "最近 (1)" })).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  for (const [locale, title] of [
    ["en", "Review inbox"],
    ["zh-CN", "复盘收件箱"],
    ["ja", "レビュー受信箱"],
    ["zh-HK", "復盤收件箱"],
    ["ko", "복기함"],
  ]) {
    await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
    await page.reload();
    await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { name: /REVIEW273B/ })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
      390,
    );
    await page.screenshot({ path: info.outputPath(`review-${locale}-390.png`), fullPage: true });
  }
  await page.evaluate(() => localStorage.setItem("tm-locale", "en"));
  await page.reload();
  const grade = page.getByRole("button", { name: "A+", exact: true });
  await grade.focus();
  await page.keyboard.press("Enter");
  await expect(grade).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "Cancel edits" }).click();
  expect(errors).toEqual([]);
});
