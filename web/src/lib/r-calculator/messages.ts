import { t as localize } from "@lingui/core/macro";
import type { WarningKey } from "./calc";

const WARNINGS: Record<WarningKey, string> = {
  get warn_stop_below_entry_long() {
    return localize({
      id: "calculator.stopMustBeBelowEntryForALong",
      message: "Stop must be below entry for a long",
    });
  },
  get warn_stop_above_entry_short() {
    return localize({
      id: "calculator.stopMustBeAboveEntryForAShort",
      message: "Stop must be above entry for a short",
    });
  },
  get warn_capital_too_small_stock() {
    return localize({
      id: "calculator.capitalCanTBuy1SharePickACheaperInstrumentOrAdd",
      message: "Capital can't buy 1 share — pick a cheaper instrument or add capital",
    });
  },
  get warn_risk_below_setting() {
    return localize({
      id: "calculator.actualRiskIsBelowYourTarget",
      message: "Actual risk is below your target.",
    });
  },
  get warn_prem_stop_vs_entry() {
    return localize({
      id: "calculator.stopPremiumMustBeBelowEntryPremium",
      message: "Stop premium must be below entry premium",
    });
  },
  get warn_capital_too_small_option() {
    return localize({
      id: "calculator.capitalCanTBuy1ContractLowerThePremiumOrAddCapital",
      message: "Capital can't buy 1 contract — lower the premium or add capital",
    });
  },
  get warn_risk_budget_too_small() {
    return localize({
      id: "calculator.riskBudgetUnder1ContractRaiseRiskOrTightenTheStop",
      message: "Risk budget under 1 contract — raise Risk% or tighten the stop",
    });
  },
  get warn_tier_sum_over100() {
    return localize({
      id: "calculator.exitTiersExceed100LowerThePercentages",
      message: "Exit tiers exceed 100% — lower the percentages",
    });
  },
  get warn_tier_rnon_positive() {
    return localize({
      id: "calculator.exitTierRMustBeGreaterThan0",
      message: "Exit tier R must be greater than 0",
    });
  },
  get warn_trailer_stop_too_high() {
    return localize({
      id: "calculator.trailingStopShouldSitBelowTheFirstTier",
      message: "Trailing stop should sit below the first tier",
    });
  },
  get warn_fvg_zone_invalid() {
    return localize({
      id: "calculator.gapTopMustSitAboveTheBottomCheckTheEdges",
      message: "Gap top must sit above the bottom — check the edges",
    });
  },
  get warn_fvg_oner_nonpositive() {
    return localize({
      id: "calculator.entryAndStopCollapseToNoRiskMoveTheEntryOrAdd",
      message: "Entry and stop collapse to no risk — move the entry or add a stop buffer",
    });
  },
  get warn_fvg_account_too_small() {
    return localize({
      id: "calculator.accountIsTooSmallToTakeEvenOneShareAtThisRisk",
      message: "Account is too small to take even one share at this risk",
    });
  },
};

export function msg(key: WarningKey): string {
  return WARNINGS[key];
}

export function limiterLabel(limiter: string): string {
  switch (limiter) {
    case "risk":
      return localize({ id: "calculator.riskLimited", message: "Risk-limited" });
    case "cash":
      return localize({ id: "calculator.cashLimited", message: "Cash-limited" });
    default:
      return "—";
  }
}
