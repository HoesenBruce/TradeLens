import { afterEach, describe, expect, it, vi } from "vite-plus/test";
import {
  DEFAULT_LOCALE,
  resolveBrowserLocale,
  getIntlLocale,
  getStoredLocale,
  isAppLocale,
  LOCALE_OPTIONS,
  navLabel,
  settingsLabel,
  setStoredLocale,
} from "./locale";

describe("locale", () => {
  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });
  it("maps app locales to Intl tags", () => {
    expect(getIntlLocale("en")).toBe("en-US");
    expect(getIntlLocale("zh-HK")).toBe("zh-HK");
    expect(getIntlLocale("ja")).toBe("ja-JP");
    expect(getIntlLocale("ko")).toBe("ko-KR");
    expect(getIntlLocale("unknown")).toBe("en-US");
  });

  it("validates supported locales", () => {
    expect(isAppLocale("en")).toBe(true);
    expect(isAppLocale("zh-HK")).toBe(true);
    expect(isAppLocale("ja")).toBe(true);
    expect(isAppLocale("ko")).toBe(true);
    expect(isAppLocale("fr")).toBe(false);
  });

  it("persists locale in localStorage", () => {
    setStoredLocale("ja");
    expect(getStoredLocale()).toBe("ja");
    setStoredLocale(DEFAULT_LOCALE);
    expect(getStoredLocale()).toBe("en");
  });

  it("translates navigation labels", () => {
    expect(navLabel("ja", "home")).toBe("ホーム");
    expect(navLabel("ko", "trades")).toBe("거래");
    expect(navLabel("zh-HK", "settings")).toBe("設定");
  });

  it("translates settings navigation labels", () => {
    expect(settingsLabel("en", "theme")).toBe("Appearance");
    expect(settingsLabel("zh-HK", "themeLight")).toBe("淺色");
    expect(settingsLabel("ja", "themeDark")).toBe("ダーク");
    expect(settingsLabel("ko", "themeSystem")).toBe("시스템");
    expect(settingsLabel("ja", "accounts")).toBe("アカウント");
    expect(settingsLabel("zh-HK", "general")).toBe("一般");
    expect(settingsLabel("en", "ai")).toBe("AI");
    expect(settingsLabel("en", "aiTitle")).toBe("AI & LLM");
    expect(settingsLabel("en", "about")).toBe("About");
    expect(settingsLabel("ja", "aboutTitle")).toBe("TraderMemos について");
  });

  it("exposes all language options", () => {
    expect(LOCALE_OPTIONS.map((o) => o.value)).toEqual(["en", "zh-CN", "zh-HK", "ja", "ko"]);
  });
});

describe("locale fallback", () => {
  it("matches browser language preferences without changing persisted values", () => {
    expect(resolveBrowserLocale(["fr-FR", "zh-Hans-SG"])).toBe("zh-CN");
    expect(resolveBrowserLocale(["zh-TW"])).toBe("zh-HK");
    expect(resolveBrowserLocale(["ja-JP", "en-US"])).toBe("ja");
    expect(resolveBrowserLocale(["de-DE"])).toBe("en");
    expect(isAppLocale("constructor")).toBe(false);
    expect(isAppLocale("__proto__")).toBe(false);
  });
  it("prefers saved settings, uses browser when absent, and rejects invalid storage", () => {
    vi.spyOn(navigator, "languages", "get").mockReturnValue(["zh-CN"]);
    localStorage.removeItem("tm-locale");
    expect(getStoredLocale()).toBe("zh-CN");
    setStoredLocale("ja");
    expect(getStoredLocale()).toBe("ja");
    localStorage.setItem("tm-locale", "unsupported");
    expect(getStoredLocale()).toBe("en");
    localStorage.clear();
    vi.restoreAllMocks();
  });
  it("translates the representative surface and falls back to English then a semantic key", () => {
    expect(navLabel("zh-CN", "home")).toBe("首页");
    expect(settingsLabel("zh-CN", "language")).toBe("语言");
    expect(settingsLabel("zh-CN", "apiTokensTitle")).toBe(settingsLabel("en", "apiTokensTitle"));
    // Simulates an untyped caller or a stale resource reference.
    expect(settingsLabel("zh-CN", "missing" as Parameters<typeof settingsLabel>[1])).toBe(
      "settings.missing",
    );
  });
});
