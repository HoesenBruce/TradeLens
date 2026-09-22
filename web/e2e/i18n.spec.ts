import { expect, test } from "@playwright/test";

// Run against a disposable real API, with service-worker caching disabled.
test.use({ serviceWorkers: "block", locale: "en-US" });
const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";

test("language switching, read-back, re-entry, persistence and safe fallbacks", async ({
  page,
  request,
}, info) => {
  const response = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(response.ok()).toBeTruthy();
  const auth = await response.json();
  await page.addInitScript(
    ({ token, api }) => {
      // No service worker support in this test context, including registration.
      Reflect.deleteProperty(Object.getPrototypeOf(navigator), "serviceWorker");
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
    },
    { token: auth.access_token, api },
  );
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const me = page.waitForResponse(
    (res) => res.url().startsWith(api) && res.url().includes("/me") && res.ok(),
  );
  await page.goto(`${web}/settings#general`, { waitUntil: "domcontentloaded" });
  await me;
  const selector = () =>
    page.getByRole("combobox", {
      name: /^(Language selector|語言選擇器|选择语言|言語セレクター)$/,
    });
  await expect(selector()).toHaveValue("en");
  for (const [locale, title, signOut, nav, exportLabel, exported, imported, failed] of [
    [
      "zh-CN",
      "常规",
      "退出登录",
      "首页",
      "导出配置",
      "已导出应用配置",
      "已导入应用配置",
      "无法导入配置",
    ],
    [
      "ja",
      "一般",
      "サインアウト",
      "ホーム",
      "設定を書き出し",
      "アプリ設定をエクスポートしました",
      "アプリ設定をインポートしました",
      "設定をインポートできませんでした",
    ],
    [
      "en",
      "General",
      "Sign out",
      "Home",
      "Export config",
      "App config exported",
      "App config imported",
      "Could not import config",
    ],
  ]) {
    await selector().selectOption(locale);
    await expect(selector()).toHaveValue(locale);
    await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: signOut, exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: nav, exact: true }).first()).toBeVisible();
    await expect.poll(() => page.evaluate(() => localStorage.getItem("tm-locale"))).toBe(locale);
    // Dismissing the immediate-apply native selector without a choice changes nothing.
    await selector().focus();
    await page.keyboard.press("Escape");
    await expect(selector()).toHaveValue(locale);
    const downloadEvent = page.waitForEvent("download");
    await page.getByRole("button", { name: exportLabel, exact: true }).click();
    await downloadEvent;
    await expect(page.getByText(exported, { exact: true })).toBeVisible();
    await page.locator('input[type="file"]').setInputFiles({
      name: "settings.json",
      mimeType: "application/json",
      buffer: Buffer.from(JSON.stringify({ locale })),
    });
    await expect(page.getByText(imported, { exact: true })).toBeVisible();
    await page.locator('input[type="file"]').setInputFiles({
      name: "invalid.json",
      mimeType: "application/json",
      buffer: Buffer.from("invalid"),
    });
    await expect(page.getByText(failed, { exact: true })).toBeVisible();
    await expect(selector()).toHaveValue(locale);
    await page.screenshot({ path: info.outputPath(`${locale}.png`), fullPage: true });
    await page.goto(`${web}/settings#about`);
    await page.goto(`${web}/settings#general`, { waitUntil: "domcontentloaded" });
    await expect(selector()).toHaveValue(locale);
    await page.reload();
    await expect(selector()).toHaveValue(locale);
    await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible();
  }
  await page.locator('input[type="file"]').setInputFiles({
    name: "chinese.json",
    mimeType: "application/json",
    buffer: Buffer.from('{"locale":"zh-CN"}'),
  });
  await expect(selector()).toHaveValue("zh-CN");
  await expect(page.getByText("已导入应用配置", { exact: true })).toBeVisible();
  for (const invalid of ["fr-FR", "constructor"]) {
    await page.evaluate((value) => localStorage.setItem("tm-locale", value), invalid);
    await page.reload();
    await expect(selector()).toHaveValue("en");
    await expect(page.locator("html")).toHaveAttribute("lang", "en-US");
  }
  // Exercise the actual development module rather than a mock translation function.
  const fallback = await page.evaluate(async () => {
    // @ts-expect-error Vite serves this source module in the E2E dev server.
    const { i18n, loadLocale } = await import("/src/i18n/index.tsx");
    await loadLocale("zh-CN");
    return [i18n._("Tz0i8g"), i18n._("issue85.missing")];
  });
  expect(fallback).toEqual(["Settings", "issue85.missing"]);
  await expect(selector()).toHaveValue("zh-CN");
  await selector().selectOption("en");
  expect(errors).toEqual([]);
});

test("first visit follows browser locale, saved preference wins", async ({ browser, request }) => {
  const response = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(response.ok()).toBeTruthy();
  const auth = await response.json();
  const context = await browser.newContext({ locale: "zh-CN", serviceWorkers: "block" });
  try {
    await context.addInitScript(
      ({ token, api }) => {
        // No service worker support in this test context, including registration.
        Reflect.deleteProperty(Object.getPrototypeOf(navigator), "serviceWorker");
        localStorage.setItem("tm_token", token);
        localStorage.setItem("tm_api_base", api);
      },
      { token: auth.access_token, api },
    );
    const page = await context.newPage();
    await page.goto(`${web}/settings#general`, { waitUntil: "domcontentloaded" });
    await expect(page.getByRole("combobox", { name: "选择语言" })).toHaveValue("zh-CN");
    await page.getByRole("combobox", { name: "选择语言" }).selectOption("en");
    await page.reload();
    await expect(page.getByRole("combobox", { name: "Language selector" })).toHaveValue("en");
    await page.evaluate(() => localStorage.setItem("tm-locale", "invalid"));
    await page.reload();
    await expect(page.getByRole("combobox", { name: "Language selector" })).toHaveValue("en");
  } finally {
    await context.close();
  }
});
