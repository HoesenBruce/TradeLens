import { afterEach, describe, expect, it } from "vite-plus/test";
import { i18n, loadLocale } from "./index";

afterEach(async () => {
  await loadLocale("en");
  localStorage.clear();
});

describe("catalog activation", () => {
  it("loads each required locale and falls back to English then a safe key", async () => {
    for (const locale of ["en", "zh-CN", "ja"]) {
      await loadLocale(locale);
      expect(i18n.locale).toBe(locale);
      expect(localStorage.getItem("tm-locale")).toBe(locale);
      expect(i18n._("issue85.missing")).toBe("issue85.missing");
    }
    await loadLocale("zh-CN");
    expect(i18n._("Tz0i8g")).toBe("Settings");
    expect(document.documentElement.lang).toBe("zh-CN");
    await loadLocale("constructor");
    expect(i18n.locale).toBe("en");
  });
  it("keeps the last requested language when catalog loads overlap", async () => {
    await Promise.all([loadLocale("ja"), loadLocale("zh-CN"), loadLocale("en")]);
    expect(i18n.locale).toBe("en");
    expect(localStorage.getItem("tm-locale")).toBe("en");
  });
});
