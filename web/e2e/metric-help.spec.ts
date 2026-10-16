import { expect, test } from "@playwright/test";

test.use({ serviceWorkers: "block" });
for (const width of [1440, 390]) {
  test(`metric help keyboard, dismissal, localization and bounds at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 844 });
    await page.goto("/e2e/fixtures/metric-help.html");
    const trigger = page.locator('[data-metric-help="holding"]');
    await trigger.focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog");
    await expect(dialog).toHaveAccessibleName("Holding duration");
    await expect(dialog).toContainText("same date = 0 days");
    await page.keyboard.press("Escape");
    await expect(dialog).not.toBeVisible();
    await expect(trigger).toBeFocused();
    await trigger.press("Space");
    await expect(dialog).toBeVisible();
    await page.getByRole("button", { name: "Outside", exact: true }).click();
    await expect(dialog).not.toBeVisible();
    for (const [locale, title] of [
      ["en", "Holding duration"],
      ["zh-CN", "持仓时长"],
      ["zh-HK", "持倉時長"],
      ["ja", "保有期間"],
      ["ko", "보유 기간"],
    ]) {
      await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
      await page.reload();
      await trigger.click();
      await expect(dialog).toHaveAccessibleName(title);
      const closeLabels: Record<string, string> = {
        en: "Close explanation",
        "zh-CN": "关闭说明",
        "zh-HK": "關閉說明",
        ja: "説明を閉じる",
        ko: "설명 닫기",
      };
      await expect(
        dialog.getByRole("button", { name: closeLabels[locale], exact: true }),
      ).toBeVisible();
      const box = await dialog.boundingBox();
      expect(box!.x).toBeGreaterThanOrEqual(0);
      expect(box!.x + box!.width).toBeLessThanOrEqual(width);
      expect(await dialog.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBeTruthy();
      await page.screenshot({
        path: info.outputPath(`${locale}-${width}.png`),
        animations: "disabled",
      });
      await dialog.getByRole("button").click();
      await expect(dialog).not.toBeVisible();
      await expect(trigger).toBeFocused();
    }
  });
}
