import type { TradeDirectionView } from "./tradeDirection";
import { t } from "@lingui/core/macro";

export function emotionLabel(value: string): string {
  const labels: Record<string, string> = {
    Calm: t({ id: "trades.calm", message: "Calm" }),
    Focused: t({ id: "trades.focused", message: "Focused" }),
    Confident: t({ id: "trades.confident", message: "Confident" }),
    Anxious: t({ id: "trades.anxious", message: "Anxious" }),
    Fearful: t({ id: "trades.fearful", message: "Fearful" }),
    Greedy: t({ id: "trades.greedy", message: "Greedy" }),
    FOMO: t({ id: "trades.fomo", message: "FOMO" }),
    Revenge: t({ id: "trades.revenge", message: "Revenge" }),
    Bored: t({ id: "trades.bored", message: "Bored" }),
    Tired: t({ id: "trades.tired", message: "Tired" }),
    Overconfident: t({ id: "trades.overconfident", message: "Overconfident" }),
  };
  return labels[value] ?? value;
}

export function sessionLabel(value: string): string {
  const labels: Record<string, string> = {
    Asia: t({ id: "trades.asia", message: "Asia" }),
    London: t({ id: "trades.london", message: "London" }),
    "New York AM": t({ id: "trades.nyAm", message: "New York AM" }),
    "New York PM": t({ id: "trades.nyPm", message: "New York PM" }),
  };
  return labels[value] ?? value;
}

export function instrumentLabel(value: string): string {
  const labels: Record<string, string> = {
    stock: t({ id: "trades.stock", message: "Stock" }),
    option: t({ id: "trades.option", message: "Option" }),
    crypto: t({ id: "trades.crypto", message: "Crypto" }),
    future: t({ id: "trades.futures", message: "Futures" }),
    futures: t({ id: "trades.futures", message: "Futures" }),
    forex: t({ id: "trades.forex", message: "Forex" }),
  };
  return labels[value] ?? value;
}

export function tradeStatusLabel(value: string): string {
  const labels: Record<string, string> = {
    WIN: t({ id: "trades.statusWin", message: "WIN" }),
    LOSS: t({ id: "trades.statusLoss", message: "LOSS" }),
    OPEN: t({ id: "trades.statusOpen", message: "OPEN" }),
    BE: t({ id: "trades.statusBe", message: "BE" }),
  };
  return labels[value] ?? value;
}

export function directionLabel(view: TradeDirectionView): string {
  switch (view.tag) {
    case "LC":
      return t({ id: "trades.longCall", message: "Long Call" });
    case "LP":
      return t({ id: "trades.longPut", message: "Long Put" });
    case "SC":
      return t({ id: "trades.shortCall", message: "Short Call" });
    case "SP":
      return t({ id: "trades.shortPut", message: "Short Put" });
    case "?":
      return t({ id: "trades.missingRight", message: "Option — call/put missing" });
    default:
      return view.long
        ? t({ id: "trades.long", message: "Long" })
        : t({ id: "trades.short", message: "Short" });
  }
}
