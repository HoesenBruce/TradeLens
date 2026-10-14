import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

// Run only with a disposable API: this suite creates synthetic identities and settings.
const api = process.env.E2E_API_URL ?? "http://localhost:8099/api/v1";
const password = "Synthetic-pr306-password-123!";
const owner = "pr306-owner@example.invalid";
const member = "pr306-member@example.invalid";
const otherOwner = "pr306-owner-b@example.invalid";
const marker = "synthetic-private-pr306";
let ownerToken: string;
let newsId: string;
let tradeId: string;
async function loginToken(request: APIRequestContext, email: string) {
  const response = await request.post(`${api}/auth/login`, { data: { email, password } });
  expect(response.ok()).toBe(true);
  return (await response.json()).access_token as string;
}
async function setSettings(request: APIRequestContext, prompt = marker) {
  const response = await request.put(`${api}/settings/ocr`, {
    headers: { Authorization: `Bearer ${ownerToken}` },
    data: {
      enabled: true,
      base_url: "https://synthetic.invalid/v1",
      model: "synthetic-model",
      api_key: "synthetic-key",
      custom_prompt: prompt,
    },
  });
  expect(response.ok()).toBe(true);
}
async function login(page: Page, email: string) {
  await page.getByRole("textbox", { name: "Username", exact: true }).fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page.getByRole("link", { name: "Settings", exact: true })).toBeVisible();
}
async function section(page: Page, id: string) {
  // Hash navigation preserves the actual application and QueryClient lifetime.
  await page.evaluate((id) => {
    window.location.hash = id;
  }, id);
}
async function signOut(page: Page) {
  await section(page, "general");
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "Username", exact: true })).toBeVisible();
}
async function noPrivateValues(page: Page) {
  expect(
    await page
      .locator("input, textarea")
      .evaluateAll((elements) =>
        elements.some((el) => (el as HTMLInputElement).value.includes("synthetic-private")),
      ),
  ).toBe(false);
  expect(await page.locator("body").innerText()).not.toContain(marker);
  expect(await page.evaluate(() => localStorage.getItem("tm-llm-model-cache"))).toBeNull();
}

test.beforeEach(async ({ page }, info) => {
  const base = info.project.metadata.apiBase as string | undefined;
  if (base) await page.addInitScript((base) => localStorage.setItem("tm_api_base", base), base);
});

test.beforeAll(async ({ request }) => {
  const status = await (await request.get(`${api}/setup/status`)).json();
  if (status.needs_setup) {
    const setup = await request.post(`${api}/setup`, { data: { email: owner, password } });
    expect(setup.ok()).toBe(true);
  }
  ownerToken = await loginToken(request, owner);
  const headers = { Authorization: `Bearer ${ownerToken}` };
  for (const [email, is_admin] of [
    [member, false],
    [otherOwner, true],
  ] as const) {
    const existing = await (await request.get(`${api}/admin/users`, { headers })).json();
    if (!existing.some((user: { email: string }) => user.email === email)) {
      expect(
        (
          await request.post(`${api}/admin/users`, { headers, data: { email, password, is_admin } })
        ).ok(),
      ).toBe(true);
    }
  }
  const news = await request.post(`${api}/news`, {
    headers,
    data: {
      title: "Synthetic manual news",
      summary: "Manual fixture",
      category: "macro",
      source: "synthetic",
      published_at: "2026-01-06T01:00:00Z",
    },
  });
  expect(news.ok()).toBe(true);
  newsId = (await news.json()).id;
  const account = await request.post(`${api}/accounts`, {
    headers,
    data: {
      name: "Synthetic account",
      broker: "manual",
      account_type: "cash",
      base_currency: "USD",
      starting_balance: 10000,
    },
  });
  expect(account.ok()).toBe(true);
  const accountId = (await account.json()).id;
  const execution = await request.post(`${api}/executions`, {
    headers,
    data: {
      account_id: accountId,
      symbol: "PR306",
      side: "buy",
      quantity: 1,
      price: 10,
      executed_at: "2026-01-06T01:00:00Z",
    },
  });
  expect(execution.ok()).toBe(true);
  tradeId = (await execution.json()).trade_id;
  expect(tradeId).toBeTruthy();
});

for (let repeat = 1; repeat <= 3; repeat++) {
  test(`owner → member → owner online and offline ${repeat}`, async ({
    page,
    context,
    request,
  }, info) => {
    await setSettings(request);
    await page.goto("/settings#ai");
    const base = info.project.metadata.apiBase as string | undefined;
    if (base) {
      await page.getByRole("button", { name: /Advanced/ }).click();
      await expect(page.getByRole("textbox", { name: "API server", exact: true })).toHaveValue(
        base,
      );
      await page.screenshot({ path: info.outputPath("prefixed-api-server.png") });
      await page.getByRole("dialog").getByRole("button", { name: "Save", exact: true }).click();
      await page.getByRole("button", { name: /Advanced/ }).click();
      await expect(page.getByRole("textbox", { name: "API server", exact: true })).toHaveValue(
        base,
      );
      await page.keyboard.press("Escape");
    }
    await login(page, owner);
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await section(page, "ai");
    await expect(page.locator("textarea")).toHaveValue(marker);
    await page.locator("textarea").fill(`${marker}-unsaved`);
    await signOut(page);
    await noPrivateValues(page);
    await login(page, member);
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await section(page, "ai");
    await noPrivateValues(page);
    const memberToken = await loginToken(request, member);
    expect(
      (
        await request.get(`${api}/settings/ocr`, {
          headers: { Authorization: `Bearer ${memberToken}` },
        })
      ).status(),
    ).toBe(403);
    await page.evaluate(async () => {
      await navigator.serviceWorker.ready;
    });
    await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
    await page.reload();
    await noPrivateValues(page);
    await context.setOffline(true);
    await section(page, "general");
    await section(page, "ai");
    await noPrivateValues(page);
    await page.reload();
    await noPrivateValues(page);
    await context.setOffline(false);
    await signOut(page);
    await login(page, owner);
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await section(page, "ai");
    await expect(page.locator("textarea")).toHaveValue(marker);
    // Offline logout is supported; offline login is not. Reconnect for the next login.
    await context.setOffline(true);
    await signOut(page);
    await noPrivateValues(page);
    await context.setOffline(false);
    await login(page, member);
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await context.setOffline(true);
    await section(page, "ai");
    await noPrivateValues(page);
    await context.setOffline(false);
  });

  test(`delayed owner request cannot populate member form ${repeat}`, async ({ page, request }) => {
    await setSettings(request);
    await page.addInitScript(() => {
      Reflect.deleteProperty(Object.getPrototypeOf(navigator), "serviceWorker");
    });
    let release!: () => void;
    let captured!: () => void;
    const ready = new Promise<void>((r) => {
      captured = r;
    });
    const gate = new Promise<void>((r) => {
      release = r;
    });
    await page.route("**/api/v1/settings/ocr", async (route) => {
      const response = await route.fetch();
      captured();
      await gate;
      await route.fulfill({ response });
    });
    await page.goto("/settings#ai");
    await login(page, owner);
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await section(page, "ai");
    await ready;
    await signOut(page);
    await login(page, member);
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await section(page, "ai");
    release();
    await expect(page.locator("textarea")).toHaveCount(0);
    await noPrivateValues(page);
  });
}

test("owner replacement and persisted model cache cannot restore previous form drafts", async ({
  page,
  request,
}) => {
  await setSettings(request);
  await page.addInitScript(() => {
    localStorage.setItem(
      "tm-llm-model-cache",
      JSON.stringify({
        state: {
          modelsByBaseUrl: { "https://synthetic-private.invalid": ["synthetic-private-model"] },
        },
        version: 0,
      }),
    );
  });
  await page.goto("/settings#ai");
  await login(page, owner);
  await page.getByRole("link", { name: "Settings", exact: true }).click();
  await section(page, "ai");
  await expect(page.locator("textarea")).toHaveValue(marker);
  await page.locator("textarea").fill(`${marker}-unsaved`);
  await signOut(page);
  await setSettings(request, "synthetic-second-owner-config");
  await login(page, otherOwner);
  await page.getByRole("link", { name: "Settings", exact: true }).click();
  await section(page, "ai");
  await expect(page.locator("textarea")).toHaveValue("synthetic-second-owner-config");
  await noPrivateValues(page);
  await section(page, "general");
  await section(page, "ai");
  await expect(page.locator("textarea")).toHaveValue("synthetic-second-owner-config");
});

const translations = [
  [
    "en",
    "AI Coach is temporarily unavailable. Rule-based review remains available.",
    "AI news analysis is temporarily unavailable. Manual news review remains available.",
  ],
  ["zh-CN", "AI 教练暂时不可用，仍可使用规则复盘。", "AI 新闻分析暂时不可用，仍可手动复盘新闻。"],
  [
    "zh-HK",
    "AI 教練暫時無法使用，仍可使用規則覆盤。",
    "AI 新聞分析暫時無法使用，仍可手動覆盤新聞。",
  ],
  [
    "ja",
    "AI コーチは現在利用できません。ルールに基づく振り返りは引き続き利用できます。",
    "AI ニュース分析は現在利用できません。手動でのニュースレビューは引き続き利用できます。",
  ],
  [
    "ko",
    "AI 코치를 일시적으로 사용할 수 없습니다. 규칙 기반 복기는 계속 사용할 수 있습니다.",
    "AI 뉴스 분석을 일시적으로 사용할 수 없습니다. 수동 뉴스 검토는 계속 사용할 수 있습니다.",
  ],
];
for (const [locale, coach, news] of translations) {
  test(`production unavailable states at narrow viewport: ${locale}`, async ({
    page,
    request,
  }, info) => {
    const token = await loginToken(request, owner);
    await page.addInitScript(
      ({ locale, token }) => {
        localStorage.setItem("tm-locale", locale);
        localStorage.setItem("tm_token", token);
      },
      { locale, token },
    );
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(message.text());
    });
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`/trades/${tradeId}`);
    await expect(page.getByText(coach, { exact: true })).toBeVisible();
    await expect(page.getByText("coach.legacyUnavailable", { exact: true })).toHaveCount(0);
    await page.screenshot({ path: info.outputPath(`${locale}-coach.png`), fullPage: true });
    await page.goto(`/news/${newsId}`);
    await expect(page.getByText(news, { exact: true })).toBeVisible();
    await expect(page.getByText("news.analysisUnavailable", { exact: true })).toHaveCount(0);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
      true,
    );
    await page.screenshot({ path: info.outputPath(`${locale}-news.png`), fullPage: true });
    expect(errors).toEqual([]);
  });
}

test("worker refuses authenticated API replay for both the same and another bearer session", async ({
  page,
  context,
  request,
}) => {
  const ownerAccess = await loginToken(request, owner);
  const memberAccess = await loginToken(request, member);
  await page.goto("/settings#ai");
  await login(page, owner);
  await page.evaluate(async () => {
    await navigator.serviceWorker.ready;
  });
  await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
  const fetchMe = (token: string) =>
    page.evaluate(async (token) => {
      try {
        const response = await fetch("/api/v1/me", {
          headers: { Authorization: `Bearer ${token}` },
        });
        return response.ok ? (await response.json()).is_admin : null;
      } catch {
        return null;
      }
    }, token);
  expect(await fetchMe(ownerAccess)).toBe(true);
  await context.setOffline(true);
  expect(await fetchMe(ownerAccess)).toBeNull();
  expect(await fetchMe(memberAccess)).toBeNull();
  expect(
    await page.evaluate(async () =>
      (await (await caches.open("api-cache")).keys()).some((key) =>
        key.url.includes("/settings/ocr"),
      ),
    ),
  ).toBe(false);
});

test("manual News cancel, re-entry, save and readback remain available", async ({
  page,
  request,
}) => {
  const token = await loginToken(request, owner);
  await page.addInitScript((token) => localStorage.setItem("tm_token", token), token);
  await page.goto(`/news/${newsId}`);
  await page.getByRole("button", { name: "Edit news thesis", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await dialog
    .getByRole("textbox", { name: "Title", exact: true })
    .fill("abandoned synthetic draft");
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await page.getByRole("button", { name: "Edit news thesis", exact: true }).click();
  await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toHaveValue(
    "Synthetic manual news",
  );
  await dialog
    .getByRole("textbox", { name: "Title", exact: true })
    .fill("Synthetic manual news saved");
  await dialog.getByRole("button", { name: "Save changes", exact: true }).click();
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Synthetic manual news saved", exact: true }),
  ).toBeVisible();
});
