import { t as tr } from "@lingui/core/macro";
import type { LucideIcon } from "lucide-react";
import { DollarSign, Euro, JapaneseYen, PoundSterling } from "lucide-react";
import type { DisplayCurrencyCode } from "./displayPrefs";

/** Lucide glyph for a display-currency code (shared ¥ for CNY/JPY). */
const CURRENCY_ICONS: Record<DisplayCurrencyCode, LucideIcon> = {
  USD: DollarSign,
  HKD: DollarSign,
  TWD: DollarSign,
  SGD: DollarSign,
  AUD: DollarSign,
  CNY: JapaneseYen,
  JPY: JapaneseYen,
  EUR: Euro,
  GBP: PoundSterling,
};

/**
 * Disambiguated symbols — five of the offered currencies are dollars and two are
 * yen, so the bare glyph cannot tell them apart in a list.
 * Pinned rather than derived because `Intl` narrow symbols collapse to `$` / `¥`
 * and its wide symbols vary by locale (SGD renders as the code in en-US).
 */
const CURRENCY_SYMBOLS: Record<DisplayCurrencyCode, string> = {
  USD: "$",
  HKD: "HK$",
  TWD: "NT$",
  CNY: "CN¥",
  EUR: "€",
  GBP: "£",
  JPY: "¥",
  AUD: "A$",
  SGD: "S$",
};

/** Issuing region — the plain-language half of the code (`HKD` → Hong Kong). */
const CURRENCY_REGIONS: Record<DisplayCurrencyCode, string> = {
  get USD() {
    return tr({ id: "currency.regionUSD", message: "United States" });
  },
  get HKD() {
    return tr({ id: "currency.regionHKD", message: "Hong Kong" });
  },
  get TWD() {
    return tr({ id: "currency.regionTWD", message: "Taiwan" });
  },
  get CNY() {
    return tr({ id: "currency.regionCNY", message: "China" });
  },
  get EUR() {
    return tr({ id: "currency.regionEUR", message: "Euro area" });
  },
  get GBP() {
    return tr({ id: "currency.regionGBP", message: "United Kingdom" });
  },
  get JPY() {
    return tr({ id: "currency.regionJPY", message: "Japan" });
  },
  get AUD() {
    return tr({ id: "currency.regionAUD", message: "Australia" });
  },
  get SGD() {
    return tr({ id: "currency.regionSGD", message: "Singapore" });
  },
};

function normalize(code: string): string {
  return code.trim().toUpperCase();
}

export function currencyIcon(code: string): LucideIcon {
  return CURRENCY_ICONS[normalize(code) as DisplayCurrencyCode] ?? DollarSign;
}

/**
 * Short symbol for a currency code. Accounts may hold a currency outside the
 * display switch (e.g. `CAD`), so unlisted codes fall back to the Intl symbol.
 */
export function currencySymbol(code: string): string {
  const key = normalize(code);
  const pinned = CURRENCY_SYMBOLS[key as DisplayCurrencyCode];
  if (pinned) return pinned;
  try {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: key,
      currencyDisplay: "symbol",
      maximumFractionDigits: 0,
    })
      .format(0)
      .replace(/[\d\s.,]/g, "");
  } catch {
    return "";
  }
}

/** Issuing region, or `undefined` for codes outside the display switch. */
export function currencyRegion(code: string): string | undefined {
  return CURRENCY_REGIONS[normalize(code) as DisplayCurrencyCode];
}
