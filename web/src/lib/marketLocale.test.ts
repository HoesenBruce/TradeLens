import { afterEach, expect, it } from "vite-plus/test";
import { loadLocale } from "@/i18n";
import { findBroker } from "./brokers";
import { formatRangeLabel } from "./dateRangePresets";
import { toolsInGroup } from "./tools";

afterEach(() => loadLocale("en"));

it("updates shared labels after language changes without translating canonical broker fields", async () => {
  const broker = findBroker("sbi")!;
  await loadLocale("zh-CN");
  expect(broker.steps[0]).toContain("成交履历");
  expect(formatRangeLabel()).toBe("全部时间");
  expect(toolsInGroup("markets")[0].label).toBe("高级图表");
  await loadLocale("ja");
  expect(broker.steps[0]).toContain("約定履歴");
  expect(formatRangeLabel()).toBe("全期間");
  expect(toolsInGroup("markets")[0].label).toBe("詳細チャート");
  expect(broker.key).toBe("sbi");
  expect(broker.accountBroker).toBe("SBI Securities");
});
