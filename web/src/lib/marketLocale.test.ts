import { afterEach, expect, it } from "vite-plus/test";
import { i18n, loadLocale } from "@/i18n";
import { findBroker } from "./brokers";
import { formatRangeLabel } from "./dateRangePresets";
import { toolsInGroup } from "./tools";

afterEach(() => loadLocale("en"));

it("updates shared labels after language changes without translating canonical broker fields", async () => {
  const broker = findBroker("sbi")!;
  await loadLocale("zh-CN");
  expect(broker.steps[0]).toContain("口座管理 → 取引履歴 → 約定履歴");
  expect(formatRangeLabel()).toBe("全部时间");
  expect(toolsInGroup("markets")[0].label).toBe("高级图表");
  await loadLocale("ja");
  expect(broker.steps[0]).toContain("約定履歴");
  expect(formatRangeLabel()).toBe("全期間");
  expect(toolsInGroup("markets")[0].label).toBe("詳細チャート");
  expect(broker.key).toBe("sbi");
  expect(broker.accountBroker).toBe("SBI Securities");
});

it("keeps SBI website labels Japanese while translating the surrounding guide", async () => {
  const broker = findBroker("sbi")!;
  const guides = [];
  for (const locale of ["en", "ja", "zh-CN"]) {
    await loadLocale(locale);
    expect(broker.formats).toBe("約定履歴 CSV · 入出金明細 CSV");
    expect(broker.steps[0]).toContain("口座管理 → 取引履歴 → 約定履歴");
    expect(broker.steps[1]).toContain("入出金 → 入出金明細");
    const hint = i18n._("imports.sbiFiles", { 0: "約定履歴 CSV", 1: "入出金明細 CSV" });
    expect(hint).toContain("約定履歴 CSV");
    expect(hint).toContain("入出金明細 CSV");
    if (locale !== "en") expect(hint).not.toContain("Account reconstruction");
    guides.push(broker.steps.join(" "));
  }
  expect(new Set(guides).size).toBe(3);
});
