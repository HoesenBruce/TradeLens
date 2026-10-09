import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";
import { PRIVACY_MASK, useDisplayPrefs } from "./displayPrefs";
import { useMoneyFormatters } from "./useMoneyFormatters";

describe("useMoneyFormatters", () => {
  afterEach(() => {
    useDisplayPrefs.setState({ privacyMode: false });
  });

  it("rebinds on a privacy flip so memoized callers re-format", () => {
    const { result } = renderHook(() => useMoneyFormatters());
    const before = result.current;
    expect(before.fmtSignedMoney(-102, "USD", "en-US")).toBe("-$102.00");
    expect(before.formatCashDisplay("withdrawal", 50, "USD")).toBe("-$50.00");

    act(() => useDisplayPrefs.getState().setPrivacyMode(true));
    const masked = result.current;
    // New identities are what make React Compiler's memo cache miss.
    expect(masked.fmtSignedMoney).not.toBe(before.fmtSignedMoney);
    expect(masked.privacy).toBe(true);
    expect(masked.fmtMoney(4182, "USD", "en-US")).toBe(PRIVACY_MASK);
    expect(masked.fmtSignedMoneyCompact(1500, "USD", "en-US")).toBe(PRIVACY_MASK);
    expect(masked.formatCashDisplay("deposit", 50, "USD")).toBe(PRIVACY_MASK);

    act(() => useDisplayPrefs.getState().setPrivacyMode(false));
    expect(result.current.fmtMoneyCompact(11790, "USD", "en-US")).toBe("$11.8K");
  });
  it("keeps every formatter masked across account currency and language changes", () => {
    const { result, rerender } = renderHook(
      ({ currency, locale }) => {
        const f = useMoneyFormatters();
        return [
          f.fmtMoney(1234, currency, locale),
          f.fmtMoneyCompact(1234, currency, locale),
          f.fmtSignedMoney(-1234, currency, locale),
          f.fmtSignedMoneyCompact(-1234, currency, locale),
          f.formatCashDisplay("withdrawal", 1234, currency),
        ];
      },
      { initialProps: { currency: "USD", locale: "en-US" } },
    );
    act(() => useDisplayPrefs.getState().setPrivacyMode(true));
    for (const currency of ["JPY", "EUR", "", "USD"]) {
      rerender({ currency, locale: "zh-CN" });
      expect(result.current).toEqual(Array(5).fill(PRIVACY_MASK));
    }
    act(() => useDisplayPrefs.getState().setPrivacyMode(false));
    expect(result.current[0]).not.toBe(PRIVACY_MASK);
  });
});
