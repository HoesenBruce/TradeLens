import { afterEach, expect, it } from "vite-plus/test";
import { cleanup, screen } from "@testing-library/react";
import { render } from "@/test/render";
import { loadLocale } from "@/i18n";
import { TradePositionTypeBadge } from "./TradePositionTypeBadge";

afterEach(async () => {
  cleanup();
  await loadLocale("en");
});

it.each([
  ["en", "Cash", "Margin", "Type unknown"],
  ["zh-CN", "现物", "信用", "类型未知"],
  ["zh-HK", "現物", "信用", "類型未知"],
  ["ja", "現物", "信用", "種別不明"],
  ["ko", "현물", "신용", "유형 알 수 없음"],
])(
  "localizes position types in %s and updates on re-entry",
  async (locale, cash, margin, unknown) => {
    await loadLocale(locale!);
    const { rerender } = render(<TradePositionTypeBadge trade={{ position_type: "cash" }} />);
    expect(screen.getByText(cash!)).toBeVisible();
    rerender(<TradePositionTypeBadge trade={{ position_type: "margin" }} />);
    expect(screen.queryByText(cash!)).not.toBeInTheDocument();
    expect(screen.getByText(margin!)).toBeVisible();
    rerender(<TradePositionTypeBadge trade={{ position_type: "unknown" }} />);
    expect(screen.queryByText(margin!)).not.toBeInTheDocument();
    expect(screen.getByText(unknown!)).toBeVisible();
  },
);

it("does not guess a position type when an older server omits it", () => {
  const { container } = render(<TradePositionTypeBadge trade={{}} />);
  expect(container).toBeEmptyDOMElement();
});
