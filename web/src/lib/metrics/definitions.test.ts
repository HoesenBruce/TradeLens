import { afterEach, expect, it } from "vite-plus/test";
import { i18n, loadLocale } from "@/i18n";
import { metricDefinitions } from "./definitions";

afterEach(async () => {
  await loadLocale("en");
});
it("localizes every definition field and interpolation in all five catalogs", async () => {
  const titles: Record<string, string> = {
    en: "Holding duration",
    "zh-CN": "持仓时长",
    "zh-HK": "持倉時長",
    ja: "保有期間",
    ko: "보유 기간",
  };
  for (const [locale, title] of Object.entries(titles)) {
    await loadLocale(locale);
    expect(i18n._(metricDefinitions.holding.title)).toBe(title);
    for (const entry of Object.values(metricDefinitions)) {
      for (const field of Object.values(entry)) {
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
