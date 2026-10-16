import { expect, test } from "@playwright/test";

// Disposable real API only. All credentials, balances and executions are synthetic.
const api = process.env.E2E_API_URL ?? "http://localhost:8092/api/v1";
test.use({ serviceWorkers: "block", viewport: { width: 1440, height: 1000 } });

test("separates amount and percentage help across locales and funding transitions", async ({
  page,
  request,
}, info) => {
  test.setTimeout(180_000);
  const credentials = { email: "issue316@synthetic.invalid", password: "synthetic-316-password" };
  await request.post(`${api}/setup`, { data: { ...credentials, name: "Synthetic help QA" } });
  const login = await request.post(`${api}/auth/login`, { data: credentials });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const account = await request.post(`${api}/accounts`, {
    headers,
    data: {
      name: "Synthetic help 329",
      broker: "Manual",
      account_type: "margin",
      base_currency: "JPY",
      starting_balance: 10000,
    },
  });
  expect(account.status()).toBe(201);
  const { id: accountId } = await account.json();
  for (const [side, price, hour] of [
    ["buy", 100, "01"],
    ["sell", 110, "02"],
  ]) {
    const fill = await request.post(`${api}/executions`, {
      headers,
      data: {
        account_id: accountId,
        symbol: "SYNTH329",
        instrument_type: "stock",
        side,
        quantity: 100,
        price,
        executed_at: `2026-10-13T${hour}:00:00Z`,
      },
    });
    expect(fill.status()).toBe(201);
  }
  await page.addInitScript(
    ({ token, accountId, api }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      if (!localStorage.getItem("tm-locale")) localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [accountId] }, version: 1 }),
      );
    },
    { token, accountId, api },
  );
  await page.goto(`/accounts/${accountId}`);
  await expect(
    page.getByRole("heading", { name: "Synthetic help 329", exact: true }),
  ).toBeVisible();
  const popup = page.locator('[data-slot="popover-popup"][data-open]');
  const titles = {
    en: [
      "Account performance",
      "Return percentage",
      "Account realized P&L",
      "Account return percentage",
    ],
    "zh-CN": ["账户绩效", "收益比例", "账户已实现盈亏", "账户收益比例"],
    "zh-HK": ["帳戶績效", "回報比例", "帳戶已實現損益", "帳戶回報比例"],
    ja: ["口座パフォーマンス", "リターン比率", "口座の実現損益", "口座リターン比率"],
    ko: ["계좌 성과", "수익 비율", "계좌 실현 손익", "계좌 수익 비율"],
  };
  const metrics = [
    "accountPerformance",
    "accountReturn",
    "accountRealizedPnl",
    "accountDetailReturn",
  ];
  const computationLabels: Record<string, string> = {
    en: "Computation",
    "zh-CN": "计算方式",
    "zh-HK": "計算方式",
    ja: "計算方法",
    ko: "계산 방법",
  };
  for (const [locale, names] of Object.entries(titles)) {
    await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
    await page.reload();
    await page.setViewportSize({ width: 390, height: 844 });
    for (const [i, metric] of metrics.entries()) {
      const trigger = page.locator(`[data-metric-help="${metric}"]`);
      await expect(trigger).toBeVisible();
      await trigger.press("Enter");
      await expect(popup).toHaveAccessibleName(names[i]);
      await expect(popup.locator("dt").first()).toHaveText(computationLabels[locale]);
      const unit = popup.locator("dl > div").nth(1).locator("dd");
      if (metric.endsWith("Return"))
        await expect(unit).toHaveText(locale === "en" || locale === "ko" ? "%." : "%。");
      else await expect(unit).not.toContainText("%");
      const box = await popup.boundingBox();
      expect(box!.x).toBeGreaterThanOrEqual(0);
      expect(box!.x + box!.width).toBeLessThanOrEqual(390);
      await page.screenshot({
        path: info.outputPath(`${locale}-${metric}.png`),
        animations: "disabled",
      });
      await page.keyboard.press("Escape");
      await expect(popup).not.toBeVisible();
      await expect(trigger).toBeFocused();
      await trigger.click();
      await expect(popup).toHaveAccessibleName(names[i]);
      await popup.locator('[data-slot="popover-close"]').click();
      await expect(trigger).toBeFocused();
    }
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    ).toBeTruthy();
  }
  await page.evaluate(() => localStorage.setItem("tm-locale", "en"));
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.reload();
  const amount = page.locator('[data-metric-help="accountPerformance"]');
  const percent = page.locator('[data-metric-help="accountReturn"]');
  await amount.click();
  await expect(popup).toHaveAccessibleName("Account performance");
  await percent.click();
  await expect(popup).toHaveAccessibleName("Return percentage");
  await amount.click();
  await expect(popup).toHaveAccessibleName("Account performance");
  await page.keyboard.press("Escape");
  const netPnl = async () => {
    const response = await request.get(`${api}/analytics/summary?account_id=${accountId}`, {
      headers,
    });
    return (await response.json()).net_pnl;
  };
  expect(await netPnl()).toBe(1000);
  // Funded → zero → negative → funded. Amount help stays available throughout.
  for (const [flow, headerReturn, detailReturn] of [
    [-10000, false, false],
    [-1000, false, true],
    [11000, true, true],
  ] as const) {
    const cash = await request.post(`${api}/cash-transactions`, {
      headers,
      data: {
        account_id: accountId,
        type: flow > 0 ? "deposit" : "withdrawal",
        amount: flow,
        currency: "JPY",
        occurred_at: "2026-10-13T00:00:00Z",
      },
    });
    expect(cash.status()).toBe(201);
    await Promise.all([
      page.waitForResponse((r) => r.url().includes("/analytics/equity-curve") && r.ok()),
      page.waitForResponse((r) => r.url().includes("/cash-transactions") && r.ok()),
      page.reload(),
    ]);
    await expect(
      page.getByRole("main").getByText("Synthetic help 329", { exact: true }),
    ).toBeVisible();
    await expect(amount).toBeVisible();
    await expect(percent).toHaveCount(headerReturn ? 1 : 0);
    await expect(page.locator('[data-metric-help="accountRealizedPnl"]')).toBeVisible();
    await expect(page.locator('[data-metric-help="accountDetailReturn"]')).toHaveCount(
      detailReturn ? 1 : 0,
    );
    if (flow === -1000) await expect(page.getByRole("main")).toContainText("-100%");
    await amount.click();
    await expect(popup).toHaveAccessibleName("Account performance");
    await page.keyboard.press("Escape");
    expect(await netPnl()).toBe(1000);
  }
});
