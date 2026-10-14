import { setupI18n } from "@lingui/core";
import { expect, it } from "vite-plus/test";
import { messages as en } from "./locales/en/messages";
import { messages as zhCN } from "./locales/zh-CN/messages";
import { messages as zhHK } from "./locales/zh-HK/messages";
import { messages as ja } from "./locales/ja/messages";
import { messages as ko } from "./locales/ko/messages";

for (const [locale, messages] of Object.entries({ en, "zh-CN": zhCN, "zh-HK": zhHK, ja, ko })) {
  it(`${locale} runtime catalog translates the security unavailable states`, () => {
    const i18n = setupI18n({ locale, messages: { [locale]: messages } });
    for (const id of ["coach.legacyUnavailable", "news.analysisUnavailable"]) {
      expect(messages[id]).toBeDefined();
      expect(i18n._(id)).not.toBe(id);
      expect(i18n._(id)).toMatch(/^(AI |AI 코치|AI 뉴스)/);
    }
  });
}
