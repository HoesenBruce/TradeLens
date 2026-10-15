import { expect, test } from "@playwright/test";

test.use({ serviceWorkers: "block", locale: "en-US" });
const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";

for (const width of [1440, 390]) {
  test(`About attribution, external links and re-entry at ${width}px`, async ({
    page,
    request,
  }, info) => {
    test.setTimeout(90_000);
    await page.setViewportSize({ width, height: 1000 });
    const login = await request.post(`${api}/auth/login`, {
      data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
    });
    expect(login.ok()).toBeTruthy();
    const { access_token: token } = await login.json();
    await page.addInitScript(
      ({ token, api }) => {
        Reflect.deleteProperty(Object.getPrototypeOf(navigator), "serviceWorker");
        localStorage.setItem("tm_token", token);
        localStorage.setItem("tm_api_base", api);
      },
      { token, api },
    );
    // Exercise external navigation without depending on GitHub availability.
    await page.context().route("https://github.com/**", (route) =>
      route.fulfill({
        contentType: "text/html",
        body: "<title>External destination</title>",
      }),
    );
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    const connected = page.waitForResponse((res) => res.url() === `${api}/system/info` && res.ok());
    await page.goto(`${web}/settings#about`);
    await connected;
    const author = page.getByRole("link", { name: "sinhong2011", exact: true });
    for (const [locale, attribution] of [
      [
        "en",
        "TradeLens is based on TraderMemos, originally developed by sinhong2011 and contributors.",
      ],
      ["zh-CN", "TradeLens 基于 TraderMemos，由 sinhong2011 和贡献者最初开发。"],
      ["zh-HK", "TradeLens 基於 TraderMemos，由 sinhong2011 及貢獻者最初開發。"],
      [
        "ja",
        "TradeLens は、sinhong2011 とコントリビューターが開発した TraderMemos を基にしています。",
      ],
      ["ko", "TradeLens는 sinhong2011 및 기여자가 개발한 TraderMemos를 기반으로 합니다."],
    ]) {
      await page.locator('a[href="#general"]').click();
      await page
        .locator("select")
        .filter({ has: page.locator('option[value="zh-CN"]') })
        .selectOption(locale);
      await page.locator('a[href="#about"]').click();
      await expect(author).toHaveCount(1);
      await expect(author.locator("..")).toHaveText(attribution);
      await expect(
        page.getByRole("heading", {
          name: /Original upstream author|原始上游作者|上流プロジェクトの原作者|업스트림 원저자/,
        }),
      ).toHaveCount(0);
      await expect(page.locator('img[src="https://github.com/sinhong2011.png"]')).toHaveCount(0);
      expect(await page.locator("body").innerText()).toContain(attribution);
      expect((await page.locator("body").innerText()).match(/sinhong2011/g)).toHaveLength(1);
      for (const href of [
        "https://github.com/sinhong2011",
        "https://github.com/sinhong2011/TraderMemos",
        "https://github.com/HoesenBruce/TradeLens/blob/main/LICENSE",
        "https://github.com/HoesenBruce/TradeLens/blob/main/NOTICE",
      ]) {
        const link = page.locator(`a[href="${href}"]`);
        await expect(link).toHaveAttribute("target", "_blank");
        await expect(link).toHaveAttribute("rel", "noopener noreferrer");
        await link.focus();
        const popupEvent = page.waitForEvent("popup");
        await page.keyboard.press("Enter");
        const popup = await popupEvent;
        await expect(popup).toHaveURL(href);
        expect(await popup.evaluate(() => window.opener === null)).toBe(true);
        await popup.close();
      }
      await author.focus();
      await page.keyboard.press("Tab");
      await expect(
        page.locator('a[href="https://github.com/sinhong2011/TraderMemos"]'),
      ).toBeFocused();
      await page.keyboard.press("Shift+Tab");
      await expect(author).toBeFocused();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
        true,
      );
      if (locale === "zh-CN") {
        await page.screenshot({ path: info.outputPath("attribution.png"), fullPage: true });
      }
      await page.reload();
      await expect(author.locator("..")).toHaveText(attribution);
    }
    expect(errors).toEqual([]);
  });
}
