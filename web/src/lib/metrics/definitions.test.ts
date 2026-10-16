import { afterEach, expect, it } from "vite-plus/test";
import { i18n, loadLocale } from "@/i18n";
import { metricDefinitions } from "./definitions";
import { messages as en } from "@/i18n/locales/en/messages";
import { messages as zhCN } from "@/i18n/locales/zh-CN/messages";
import { messages as zhHK } from "@/i18n/locales/zh-HK/messages";
import { messages as ja } from "@/i18n/locales/ja/messages";
import { messages as ko } from "@/i18n/locales/ko/messages";

afterEach(async () => {
  await loadLocale("en");
});
it("localizes every definition field and interpolation in all five catalogs", async () => {
  const catalogs: Record<string, typeof en> = { en, "zh-CN": zhCN, "zh-HK": zhHK, ja, ko };
  const titles: Record<string, string> = {
    en: "Holding duration",
    "zh-CN": "持仓时长",
    "zh-HK": "持倉時長",
    ja: "保有期間",
    ko: "보유 기간",
  };
  await loadLocale("en");
  const english = Object.fromEntries(
    Object.entries(metricDefinitions).map(([id, entry]) => [id, i18n._(entry.caveat)]),
  );
  for (const [locale, title] of Object.entries(titles)) {
    await loadLocale(locale);
    expect(i18n._(metricDefinitions.holding.title)).toBe(title);
    for (const [id, entry] of Object.entries(metricDefinitions)) {
      expect(i18n._(entry.caveat) === english[id], `${locale}.${id}`).toBe(locale === "en");
      for (const field of Object.values(entry)) {
        expect(Object.hasOwn(catalogs[locale], field.id!)).toBe(true);
        expect(i18n._(field)).not.toBe(field.id);
        expect(i18n._(field).length).toBeGreaterThan(0);
      }
    }
    expect(i18n._("metric.help", { title })).toContain(title);
    expect(i18n._("metric.help", { title })).not.toContain("{title}");
  }
  await loadLocale("unsupported");
  expect(i18n._(metricDefinitions.holding.title)).toBe("Holding duration");
  i18n.load("ko", { "metric.unit": "" });
  await loadLocale("ko");
  expect(i18n._("metric.unit")).toBe("단위");
});
