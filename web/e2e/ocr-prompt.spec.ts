/// <reference types="node" />
import { expect, test, type Locator } from "@playwright/test";

test.use({ serviceWorkers: "block", locale: "en-US" });
const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";
const prompt = `{
  "instruction": "保留缩进、日本語、한국어 and internal whitespace",
  "unbroken": "${"ABCDEFGHIJ".repeat(300)}",
  "rows": [
${Array.from({ length: 40 }, (_, i) => `    {"index": ${i}, "side": "buy"}`).join(",\n")}
  ]
}`;

async function expectContained(input: Locator) {
  const bounds = await input.evaluate((el) => {
    const textarea = el as HTMLTextAreaElement;
    const rect = el.getBoundingClientRect();
    const wrapper = el.parentElement!.getBoundingClientRect();
    const form = el.closest("form")!;
    return {
      height: rect.height,
      minHeight: Number.parseFloat(getComputedStyle(el).minHeight),
      inside:
        rect.left >= wrapper.left &&
        rect.right <= wrapper.right &&
        rect.bottom <= wrapper.bottom &&
        rect.left >= 0 &&
        rect.right <= innerWidth,
      noHorizontalScroll: textarea.scrollWidth <= textarea.clientWidth + 1,
      scrolls: textarea.scrollHeight > textarea.clientHeight,
      actionsBelow: [...form.querySelectorAll("button")]
        .filter((button) =>
          /^(Test|Save|测试|保存|測試|儲存|テスト|저장|테스트)$/.test(button.textContent ?? ""),
        )
        .every((button) => button.getBoundingClientRect().top > rect.bottom),
    };
  });
  expect(bounds.minHeight).toBeGreaterThanOrEqual(180);
  expect(bounds.height).toBeGreaterThanOrEqual(180);
  expect(bounds.height).toBeLessThanOrEqual(384);
  expect(bounds).toMatchObject({
    inside: true,
    noHorizontalScroll: true,
    scrolls: true,
    actionsBelow: true,
  });
}

// Use a disposable API instance: this test saves the owner's OCR settings.
for (const width of [1440, 768, 390, 320]) {
  test(`OCR prompt containment, editing and persistence at ${width}px`, async ({
    page,
    request,
    browserName,
  }, info) => {
    test.setTimeout(90_000);
    await page.setViewportSize({ width, height: 1000 });
    const login = await request.post(`${api}/auth/login`, {
      data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
    });
    expect(login.ok()).toBeTruthy();
    const { access_token: token } = await login.json();
    const headers = { Authorization: `Bearer ${token}` };
    const settings = await request.get(`${api}/settings/ocr`, { headers });
    expect(settings.ok()).toBeTruthy();
    const original = await settings.json();
    const seed = await request.put(`${api}/settings/ocr`, {
      headers,
      data: { ...original, enabled: true, custom_prompt: "" },
    });
    expect(seed.ok()).toBeTruthy();
    await page.addInitScript(
      ({ token, api }) => {
        Reflect.deleteProperty(Object.getPrototypeOf(navigator), "serviceWorker");
        localStorage.setItem("tm_token", token);
        localStorage.setItem("tm_api_base", api);
        localStorage.setItem("tm-locale", "en");
      },
      { token, api },
    );
    await page.goto(`${web}/settings#ai`);
    const input = page.getByRole("textbox", { name: "Custom prompt", exact: true });
    const save = page.getByRole("button", { name: "Save", exact: true });
    await expect(input).toBeEnabled();
    await expect(input).toHaveValue("");
    expect((await input.getAttribute("placeholder"))!.length).toBeGreaterThan(1000);
    await expectContained(input);
    await input.scrollIntoViewIfNeeded();
    await page.screenshot({ path: info.outputPath("default-prompt.png"), fullPage: true });
    await input.fill("short prompt");
    expect((await input.boundingBox())!.height).toBeGreaterThanOrEqual(180);
    await input.fill(prompt);
    await expectContained(input);
    await input.press("ControlOrMeta+End");
    await input.press("Shift+ArrowLeft");
    expect(
      await input.evaluate(
        (el) =>
          (el as HTMLTextAreaElement).selectionEnd - (el as HTMLTextAreaElement).selectionStart,
      ),
    ).toBe(1);
    await input.press("}");
    await expect(input).toHaveValue(prompt);
    // Safari uses Option-Tab to include buttons in keyboard navigation.
    await input.press(browserName === "webkit" ? "Alt+Tab" : "Tab");
    await expect(page.getByRole("button", { name: "Test", exact: true })).toBeFocused();
    const tested = page.waitForResponse(
      (res) => res.url() === `${api}/settings/ocr/test` && res.ok(),
    );
    await page.keyboard.press("Enter");
    expect(await (await tested).json()).toMatchObject({ ok: true });
    await expect(input).toHaveValue(prompt);
    await expectContained(input);
    await save.click();
    await expect(page.getByText("Settings updated.", { exact: true })).toBeVisible();
    await page.reload();
    await expect(input).toHaveValue(prompt);
    const readBack = await request.get(`${api}/settings/ocr`, { headers });
    expect(await readBack.json()).toMatchObject({
      custom_prompt: prompt,
      base_url: original.base_url,
      model: original.model,
      api_key_set: original.api_key_set,
    });
    await input.scrollIntoViewIfNeeded();
    await page.screenshot({ path: info.outputPath("custom-prompt.png"), fullPage: true });
    // Abandon an edit, then re-enter: only the saved prompt should return.
    await input.fill("abandoned draft");
    await page.locator('a[href="#general"]').click();
    await expect(input).toHaveCount(0);
    await page.locator('a[href="#ai"]').click();
    await expect(input).toHaveValue(prompt);
    await input.fill("");
    await save.click();
    await expect(page.getByText("Settings updated.", { exact: true })).toBeVisible();
    await page.reload();
    await expect(input).toHaveValue("");
    await expectContained(input);
    // Localized labels must fit the same narrow layout.
    for (const locale of ["zh-CN", "zh-HK", "ja", "ko", "en"]) {
      await page.locator('a[href="#general"]').click();
      await page
        .locator("select")
        .filter({ has: page.locator('option[value="zh-CN"]') })
        .selectOption(locale);
      await page.locator('a[href="#ai"]').click();
      await expectContained(page.locator("textarea"));
      if (locale === "ja")
        await page.screenshot({ path: info.outputPath("localized-prompt.png"), fullPage: true });
    }
    // Exercise disabled/re-enabled controls without changing persisted settings.
    const enabled = page.getByRole("switch", { name: "Enabled", exact: true });
    await enabled.click();
    await expect(input).toBeDisabled();
    await enabled.click();
    await expect(input).toBeEnabled();
  });
}
