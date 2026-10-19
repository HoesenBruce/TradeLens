import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it } from "vite-plus/test";
import { i18n, loadLocale } from "@/i18n";
import { render } from "@/test/render";
import { SBIImportGuide } from "./SBIImportGuide";

afterEach(() => loadLocale("en"));

it("localizes all guide copy and preserves official labels in all five locales", async () => {
  const user = userEvent.setup();
  render(<SBIImportGuide />);
  const trigger = screen.getByRole("button");
  trigger.focus();
  await user.keyboard("{Enter}");
  expect(trigger).toHaveAttribute("aria-expanded", "true");
  const prose = new Map<string, Set<string>>();
  for (const locale of ["en", "ja", "zh-CN", "zh-HK", "ko"]) {
    await act(() => loadLocale(locale));
    expect(screen.getByText("口座管理 → 取引履歴 → 約定履歴")).toBeVisible();
    expect(screen.getByText("入出金 → 入出金明細")).toBeVisible();
    expect(
      screen.getByText("My資産 → 実現損益詳細 → 国内株式（現物） / 国内株式（信用）"),
    ).toBeVisible();
    expect(screen.getByText(/売却 \/ 返済売 \/ 返済買 \/ 現渡/)).toBeVisible();
    expect(screen.getByRole("link")).toHaveAttribute(
      "href",
      "https://search.sbisec.co.jp/v2/popwin/guide/tool/assets/03_profits_loss/profits_loss_domestic.html",
    );
    for (const key of [
      "sbiGuide",
      "sbiExecutionsPurpose",
      "sbiExecutionsExport",
      "sbiCashPurpose",
      "sbiCashExport",
      "sbiPnlPurpose",
      "sbiPnlExport",
      "sbiPnlPeriod",
      "sbiPnlCaveats",
      "sbiPnlValues",
      "sbiOfficialHelp",
      "sbiPreviewDuplicates",
      "sbiHistoryGuide",
    ]) {
      const id = `imports.${key}`;
      expect(i18n._(id)).not.toBe(id);
      const versions = prose.get(id) ?? new Set<string>();
      versions.add(i18n._(id));
      prose.set(id, versions);
    }
  }
  for (const versions of prose.values()) expect(versions.size).toBe(5);
  trigger.focus();
  await user.keyboard(" ");
  expect(trigger).toHaveAttribute("aria-expanded", "false");
});
