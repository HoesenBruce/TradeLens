import { describe, expect, it } from "vite-plus/test";
import { createElement, type ReactNode } from "react";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useDisplayPrefs } from "../displayPrefs";
import { useMoneyFx, convertAmount } from "./useMoneyFx";

describe("convertAmount", () => {
  it("multiplies by FX rate", () => {
    expect(convertAmount(100, 7.8)).toBeCloseTo(780);
    expect(convertAmount(-50, 7.8)).toBeCloseTo(-390);
  });
});

it("converts supported source currencies once and leaves unknown scopes unresolved", () => {
  const client = new QueryClient();
  client.setQueryData(["fx-rate", "USD", "JPY"], { rate: 150 });
  client.setQueryData(["fx-rate", "JPY", "USD"], { rate: 1 / 150 });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
  useDisplayPrefs.setState({ displayCurrency: "JPY" });
  const jpy = renderHook(() => useMoneyFx("JPY"), { wrapper });
  expect(jpy.result.current.toDisplay(8362055)).toBe(8362055);
  const usd = renderHook(() => useMoneyFx("USD"), { wrapper });
  expect(usd.result.current.toDisplay(100)).toBe(15000);
  const unknown = renderHook(() => useMoneyFx(""), { wrapper });
  expect(unknown.result.current.currency).toBe("");
  expect(unknown.result.current.toDisplay(100)).toBe(100);
  jpy.unmount();
  usd.unmount();
  unknown.unmount();
  useDisplayPrefs.setState({ displayCurrency: "USD" });
  const reverse = renderHook(() => useMoneyFx("JPY"), { wrapper });
  expect(reverse.result.current.toDisplay(15000)).toBe(100);
  reverse.unmount();
  useDisplayPrefs.setState({ displayCurrency: null });
});
