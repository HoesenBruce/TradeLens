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
    await loadLocale("zh-HK");
    expect(i18n._("auth.showValue")).toBe("Show value");
    expect(document.documentElement.lang).toBe("zh-HK");
    await loadLocale("constructor");
    expect(i18n.locale).toBe("en");
  });
  it("keeps the last requested language when catalog loads overlap", async () => {
    await Promise.all([loadLocale("ja"), loadLocale("zh-CN"), loadLocale("en")]);
    expect(i18n.locale).toBe("en");
    expect(localStorage.getItem("tm-locale")).toBe("en");
  });
});

it("translates news labels and interpolated values in all required locales", async () => {
  for (const [locale, heading, period, current] of [
    ["en", "News thesis", "5D", "Current: QA"],
    ["zh-CN", "新闻观点", "5个交易日", "当前：QA"],
    ["ja", "ニュース仮説", "5取引日", "現在：QA"],
  ]) {
    await loadLocale(locale);
    expect(i18n._("news.heading")).toBe(heading);
    expect(i18n._("news.days", { h: 5 })).toBe(period);
    expect(i18n._("news.current", { 0: "QA" })).toBe(current);
  }
});

it("preserves trade label interpolation in Chinese and Japanese", async () => {
  for (const [locale, amount, field] of [
    ["zh-CN", "金额（JPY）", "分红金额 2"],
    ["ja", "金額（JPY）", "配当額 2"],
  ]) {
    await loadLocale(locale);
    expect(i18n._("trades.currencyAmount", { currency: "JPY" })).toBe(amount);
    expect(i18n._("trades.dividendAmountField", { suffix: " 2" })).toBe(field);
  }
});

it("refreshes secondary labels without changing broker, rule or hotkey identifiers", async () => {
  const { BROKERS } = await import("@/lib/brokers");
  const { currencyRegion } = await import("@/lib/currency");
  const { hotkeyCommandName } = await import("@/lib/keybindings");
  const { RISK_RULE_DEFS } = await import("@/lib/settingsFormSchema");
  const broker = BROKERS.find((b) => b.key === "ibkr")!;
  const ids = { broker: broker.accountBroker, rule: RISK_RULE_DEFS[0].key };
  for (const [locale, reports, region] of [
    ["en", "Reports", "Japan"],
    ["zh-CN", "报表", "日本"],
    ["ja", "レポート", "日本"],
    ["ko", "Reports", "Japan"],
    ["zh-HK", "Reports", "Japan"],
  ]) {
    await loadLocale(locale);
    expect(hotkeyCommandName("nav-stats")).toBe(reports);
    expect(currencyRegion("JPY")).toBe(region);
    expect({ broker: broker.accountBroker, rule: RISK_RULE_DEFS[0].key }).toEqual(ids);
    expect(broker.steps.join(" ")).toContain("Flex Queries");
  }
});

it("translates review actions in every supported locale", async () => {
  for (const [locale, title, save] of [
    ["en", "Review inbox", "Save and next"],
    ["zh-CN", "复盘收件箱", "保存并进入下一笔"],
    ["zh-HK", "復盤收件箱", "儲存並進入下一筆"],
    ["ja", "レビュー受信箱", "保存して次へ"],
    ["ko", "복기함", "저장 후 다음"],
  ]) {
    await loadLocale(locale);
    expect(i18n._("review.title")).toBe(title);
    expect(i18n._("review.save")).toBe(save);
  }
});
