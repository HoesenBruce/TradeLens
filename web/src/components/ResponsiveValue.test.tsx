import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vite-plus/test";
import { ResponsiveValue } from "./ResponsiveValue";
import { TooltipProvider } from "./ui/tooltip";
import { useDisplayPrefs, PRIVACY_MASK } from "@/lib/displayPrefs";
import { useMoneyFormatters } from "@/lib/useMoneyFormatters";

let available = 200;
let textWidth = 180;
let resize: () => void;
const disconnect = vi.fn<() => void>();

beforeEach(() => {
  available = 200;
  textWidth = 180;
  resize = undefined as never;
  useDisplayPrefs.setState({ privacyMode: false });
  vi.spyOn(HTMLElement.prototype, "clientWidth", "get").mockImplementation(() => available);
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
    () => ({ width: textWidth }) as DOMRect,
  );
  vi.stubGlobal(
    "ResizeObserver",
    class {
      constructor(callback: () => void) {
        resize ??= callback;
      }
      observe() {}
      disconnect = disconnect;
    },
  );
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it("measures the full amount with a margin, stays compact and restores it on widening", () => {
  const { container, unmount } = render(
    <TooltipProvider>
      <ResponsiveValue value="$1,000,000.00" compactValue="$1M" />
    </TooltipProvider>,
  );
  const amount = container.querySelector("[data-money-format]")!;
  expect(amount).toHaveAttribute("data-money-format", "full");
  expect(amount).not.toHaveAttribute("tabindex", "0");
  available = 182;
  act(() => resize());
  expect(amount).toHaveAttribute("data-money-format", "compact");
  expect(amount).toHaveAttribute("aria-label", "$1,000,000.00");
  expect(amount).toHaveAttribute("tabindex", "0");
  act(() => resize());
  expect(amount).toHaveAttribute("data-money-format", "compact");
  expect(screen.getByText("$1M")).toBeVisible();
  available = 200;
  act(() => resize());
  expect(amount).toHaveAttribute("data-money-format", "full");
  expect(amount).not.toHaveAttribute("aria-label");
  unmount();
  expect(disconnect).toHaveBeenCalled();
});

function Money({ amount, locale = "en-US" }: { amount: number; locale?: string }) {
  const { fmtSignedMoney, fmtSignedMoneyCompact } = useMoneyFormatters();
  return (
    <ResponsiveValue
      value={fmtSignedMoney(amount, "USD", locale)}
      compactValue={fmtSignedMoneyCompact(amount, "USD", locale)}
    />
  );
}

it("uses existing localized signed formatters and updates values without resizing", () => {
  available = 100;
  const { container, rerender } = render(
    <TooltipProvider>
      <Money amount={-10000} />
    </TooltipProvider>,
  );
  expect(screen.getByText("-$10K")).toBeVisible();
  rerender(
    <TooltipProvider>
      <Money amount={1000000} locale="zh-CN" />
    </TooltipProvider>,
  );
  expect(screen.getByText("+US$100万")).toBeVisible();
  textWidth = 30;
  rerender(
    <TooltipProvider>
      <Money amount={0} />
    </TooltipProvider>,
  );
  expect(container.querySelector("[data-money-format]")).toHaveAttribute(
    "data-money-format",
    "full",
  );
});

it("removes full and compact amounts from the DOM and accessible label when privacy changes", () => {
  available = 100;
  const { container } = render(
    <TooltipProvider>
      <Money amount={1000000} />
    </TooltipProvider>,
  );
  expect(screen.getByText("+$1M")).toBeVisible();
  act(() => useDisplayPrefs.getState().setPrivacyMode(true));
  expect(container.textContent).not.toContain("1,000,000");
  expect(container.textContent).not.toContain("1M");
  expect(container.querySelector("[data-money-format]")).toHaveAttribute(
    "aria-label",
    PRIVACY_MASK,
  );
  act(() => useDisplayPrefs.getState().setPrivacyMode(false));
  expect(screen.getByText("+$1M")).toBeVisible();
});

it("reveals the full amount on hover and remasks an already open tooltip", async () => {
  available = 100;
  const user = userEvent.setup();
  render(
    <TooltipProvider>
      <Money amount={1000000} />
    </TooltipProvider>,
  );
  const amount = screen.getByLabelText("+$1,000,000.00");
  await user.hover(amount);
  await waitFor(() =>
    expect(document.querySelector('[data-slot="tooltip-popup"]')).toHaveTextContent(
      "+$1,000,000.00",
    ),
  );
  await act(async () => useDisplayPrefs.getState().setPrivacyMode(true));
  expect(document.querySelector('[data-slot="tooltip-popup"]')).toHaveTextContent(PRIVACY_MASK);
  expect(document.body.textContent).not.toContain("1,000,000");
  await user.unhover(amount);
  await waitFor(() =>
    expect(document.querySelector('[data-slot="tooltip-popup"]')).not.toBeInTheDocument(),
  );
});

it("supports click and keyboard disclosure, Escape, and closes when the value fits again", async () => {
  available = 100;
  const user = userEvent.setup();
  render(
    <TooltipProvider>
      <ResponsiveValue value="-$1,000,000.00" compactValue="-$1M" />
    </TooltipProvider>,
  );
  const amount = screen.getByRole("button", { name: "-$1,000,000.00" });
  await user.click(amount);
  await waitFor(() => expect(screen.getByRole("tooltip")).toHaveTextContent("-$1,000,000.00"));
  await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument());
  await user.keyboard("{Enter}");
  await waitFor(() => expect(screen.getByRole("tooltip")).toBeVisible());
  await user.keyboard("[Space]");
  await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument());
  await user.keyboard("[Space]");
  await waitFor(() => expect(screen.getByRole("tooltip")).toBeVisible());
  available = 300;
  act(() => resize());
  await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument());
  expect(amount).not.toHaveAttribute("tabindex", "0");
});

it("does not introduce another control inside a navigable day card", () => {
  available = 100;
  render(
    <TooltipProvider>
      <button aria-label="Day: -$1,000,000.00">
        <ResponsiveValue value="-$1,000,000.00" compactValue="-$1M" disclosure={false} />
      </button>
    </TooltipProvider>,
  );
  expect(screen.getAllByRole("button")).toHaveLength(1);
  expect(screen.getByText("-$1M")).not.toHaveAttribute("tabindex");
});

it("fits long localized abbreviations without shrinking below a readable 18px", () => {
  available = 100;
  vi.spyOn(window, "getComputedStyle").mockReturnValue({ fontSize: "38px" } as CSSStyleDeclaration);
  const { container } = render(
    <TooltipProvider>
      <ResponsiveValue value="+JP¥12,345,678" compactValue="+JP¥1234.6万" />
    </TooltipProvider>,
  );
  const visible = container.querySelector<HTMLElement>("[data-money-format] > span:last-child")!;
  expect(Number.parseFloat(visible.style.fontSize)).toBeGreaterThanOrEqual(18);
  expect(Number.parseFloat(visible.style.fontSize)).toBeLessThan(38);
  available = 40;
  act(() => resize());
  expect(visible.style.fontSize).toBe("18px");
  available = 400;
  act(() => resize());
  expect(visible.style.fontSize).toBe("");
});
