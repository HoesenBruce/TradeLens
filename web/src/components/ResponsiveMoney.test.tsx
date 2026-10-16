import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vite-plus/test";
import { ResponsiveMoney } from "./ResponsiveMoney";
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
  useDisplayPrefs.setState({ privacyMode: false });
  vi.spyOn(HTMLElement.prototype, "clientWidth", "get").mockImplementation(() => available);
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
    () => ({ width: textWidth }) as DOMRect,
  );
  vi.stubGlobal(
    "ResizeObserver",
    class {
      constructor(callback: () => void) {
        resize = callback;
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
      <ResponsiveMoney value="$1,000,000.00" compactValue="$1M" />
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
    <ResponsiveMoney
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
