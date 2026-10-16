import type { MessageDescriptor } from "@lingui/core";
import { msg } from "@lingui/core/macro";

export interface MetricDefinition {
  title: MessageDescriptor;
  definition: MessageDescriptor;
  formula: MessageDescriptor;
  unit: MessageDescriptor;
  caveat: MessageDescriptor;
  documentation?: { href: string; label: MessageDescriptor };
}

// Stable IDs identify explanations, never computations or user data.
export const metricDefinitions = {
  holding: {
    title: msg({ id: "metric.holding.title", message: "Holding duration" }),
    definition: msg({
      id: "metric.holding.definition",
      message: "Time from the recorded opening to the close, or to now for an open trade.",
    }),
    formula: msg({
      id: "metric.holding.formula",
      message:
        "Precise endpoints: elapsed time, rounded to minutes for display (under one minute: <1m). Any date-only endpoint: difference between exchange-local calendar dates; same date = 0 days, next date = 1 day.",
    }),
    unit: msg({
      id: "metric.holding.unit",
      message: "Days, hours and minutes; calendar days for date-only records.",
    }),
    caveat: msg({
      id: "metric.holding.caveat",
      message:
        "Date-only SBI or CSV fills do not establish intraday holding time. Missing opening history, unknown precision or invalid boundaries show —. The API supplies the market timezone and precision; the viewer timezone does not change calendar days.",
    }),
  },
  quantities: {
    title: msg({ id: "metric.quantities.title", message: "Opened, closed and remaining quantity" }),
    definition: msg({
      id: "metric.quantities.definition",
      message:
        "Cumulative opened quantity and the current remaining exposure in instrument units, supplied by trade.quantities.",
    }),
    formula: msg({
      id: "metric.quantities.formula",
      message:
        "Closed = opened − remaining. Close progress = closed ÷ opened × 100, bounded to 0–100%; remaining % = 100 − close progress.",
    }),
    unit: msg({
      id: "metric.quantities.unit",
      message: "Shares, contracts or instrument units; %.",
    }),
    caveat: msg({
      id: "metric.quantities.caveat",
      message:
        "Long and short use positive magnitudes. Reversals belong to separate trade lifecycles. Approved corporate actions use normalized units. Missing opening history, unverified units or inconsistent inventory show —, never zero. These are quantities, not current market value.",
    }),
  },
  pnl: {
    title: msg({ id: "metric.pnl.title", message: "Gross and net realized P&L" }),
    definition: msg({
      id: "metric.pnl.definition",
      message:
        "Recorded trading results before and after fees; excludes unrealized mark-to-market and separately linked dividends.",
    }),
    formula: msg({
      id: "metric.pnl.formula",
      message:
        "Default closed trade: signed (weighted exit − weighted entry) × closed quantity × multiplier; net = gross − fees. Summaries sum eligible closed-trade amounts.",
    }),
    unit: msg({
      id: "metric.pnl.unit",
      message: "Trade P&L currency; monetary results usually rounded to 2 decimals before display.",
    }),
    caveat: msg({
      id: "metric.pnl.caveat",
      message:
        "SBI uses its existing settlement/average-cost path and may report realized amounts on partial closes. Default open trades keep P&L unavailable until fully closed; summaries exclude open trades. Missing basis or settlement evidence stays —. Total including dividends is a separate metric.",
    }),
  },
  fees: {
    title: msg({ id: "metric.fees.title", message: "Fees and commissions" }),
    definition: msg({
      id: "metric.fees.definition",
      message: "Recorded fees allocated to the trade or eligible closed-trade sample.",
    }),
    formula: msg({
      id: "metric.fees.formula",
      message:
        "Trade: allocated execution fees + commissions. Detail fee drag = fees ÷ absolute gross P&L; absent at zero/unknown gross. Reports fee % = total fees ÷ (winning net amounts + absolute losing net amounts) × 100; zero denominator gives 0%.",
    }),
    unit: msg({ id: "metric.fees.unit", message: "P&L currency; fee drag in %." }),
    caveat: msg({
      id: "metric.fees.caveat",
      message:
        "Reversal fees follow existing allocation; SBI uses recorded settlement fees where applicable. Missing source charges are not estimates. Reports use absolute net movement as their fee-% base, not turnover or invested capital.",
    }),
  },
  tradeReturn: {
    title: msg({ id: "metric.tradeReturn.title", message: "Trade P&L percent" }),
    definition: msg({
      id: "metric.tradeReturn.definition",
      message:
        "The API-provided realized return relative to the trade entry basis, not account capital.",
    }),
    formula: msg({
      id: "metric.tradeReturn.formula",
      message:
        "Default closed trade: net P&L ÷ (unrounded weighted entry × closed quantity × multiplier) × 100. SBI settlement overrides use recorded average entry × opened quantity. Results round to 2 decimals.",
    }),
    unit: msg({ id: "metric.tradeReturn.unit", message: "%." }),
    caveat: msg({
      id: "metric.tradeReturn.caveat",
      message:
        "The denominator follows the existing broker path, not remaining exposure. Default zero basis returns 0; unavailable P&L stays —. Displayed rounded prices may not reproduce the exact percent. Dividends are excluded.",
    }),
  },
  plannedRR: {
    title: msg({ id: "metric.plannedRR.title", message: "Planned reward / risk" }),
    definition: msg({
      id: "metric.plannedRR.definition",
      message: "The price-distance ratio of the recorded target and stop, before fees.",
    }),
    formula: msg({
      id: "metric.plannedRR.formula",
      message:
        "Long: (target − entry) ÷ (entry − stop). Short: (entry − target) ÷ (stop − entry). Entry is the recorded average entry price.",
    }),
    unit: msg({ id: "metric.plannedRR.unit", message: "Ratio, displayed as reward:1." }),
    caveat: msg({
      id: "metric.plannedRR.caveat",
      message:
        "Requires both target and stop and a positive risk distance; otherwise —. Quantity, multiplier, fees and journal initial risk do not enter this ratio. A target on the adverse side can give a negative reward.",
    }),
  },
  realizedR: {
    title: msg({ id: "metric.realizedR.title", message: "Realized R" }),
    definition: msg({
      id: "metric.realizedR.definition",
      message: "Recorded net trading P&L relative to journal initial risk.",
    }),
    formula: msg({ id: "metric.realizedR.formula", message: "Net P&L ÷ initial risk." }),
    unit: msg({
      id: "metric.realizedR.unit",
      message: "R (risk multiples), displayed to 2 decimals.",
    }),
    caveat: msg({
      id: "metric.realizedR.caveat",
      message:
        "Missing/zero risk or unavailable P&L gives —. The list requires positive risk; the detail API accepts nonzero risk. Dividends are excluded. This is distinct from the target/stop planned ratio.",
    }),
  },
  winRate: {
    title: msg({ id: "metric.winRate.title", message: "Win rate" }),
    definition: msg({
      id: "metric.winRate.definition",
      message: "Share of eligible closed trades with positive net P&L after fees.",
    }),
    formula: msg({
      id: "metric.winRate.formula",
      message: "Wins ÷ (wins + losses + breakeven) × 100. A trade counts once, not once per fill.",
    }),
    unit: msg({ id: "metric.winRate.unit", message: "%; API stores a ratio from 0 to 1." }),
    caveat: msg({
      id: "metric.winRate.caveat",
      message:
        "Breakeven is included in the denominator. Open/partially closed trades and unknown P&L are excluded from summary counts even when partial realized P&L is available. No eligible trades gives 0%. Account/date filters define the sample.",
    }),
  },
  payoff: {
    title: msg({ id: "metric.payoff.title", message: "Average win / loss ratio" }),
    definition: msg({
      id: "metric.payoff.definition",
      message: "Relative size of winning and losing net outcomes in the closed-trade sample.",
    }),
    formula: msg({
      id: "metric.payoff.formula",
      message:
        "Mean mode: average positive net P&L ÷ average absolute negative net P&L. Median mode uses the corresponding medians.",
    }),
    unit: msg({ id: "metric.payoff.unit", message: "Multiple, displayed with ×." }),
    caveat: msg({
      id: "metric.payoff.caveat",
      message:
        "The API rounds averages/medians to 2 decimals before this ratio. No losses but a positive win displays ∞; neither or a zero ratio displays —. Breakeven is in neither group. Net/gross display mode does not reclassify this ratio.",
    }),
  },
  profitFactor: {
    title: msg({ id: "metric.profitFactor.title", message: "Profit factor" }),
    definition: msg({
      id: "metric.profitFactor.definition",
      message:
        "Winning net amounts relative to absolute losing net amounts for eligible closed trades.",
    }),
    formula: msg({
      id: "metric.profitFactor.formula",
      message: "Sum of positive net P&L ÷ absolute sum of negative net P&L, rounded to 2 decimals.",
    }),
    unit: msg({ id: "metric.profitFactor.unit", message: "Ratio; no currency." }),
    caveat: msg({
      id: "metric.profitFactor.caveat",
      message:
        "Despite the API names gross_profit/gross_loss, these are after-fee net buckets. With no losses the API returns 0, not infinity; the header shows 0.00 and some cards show —. Breakeven adds to neither sum. Net/gross display mode does not change summary profit factor.",
    }),
  },
  drawdown: {
    title: msg({ id: "metric.drawdown.title", message: "Maximum drawdown amount" }),
    definition: msg({
      id: "metric.drawdown.definition",
      message: "Largest recorded peak-to-trough decline of the API equity curve.",
    }),
    formula: msg({
      id: "metric.drawdown.formula",
      message:
        "Equity starts at 0 and adds signed cash-ledger flows and eligible closed-trade net P&L chronologically. Max drawdown = maximum (running peak − equity), rounded to 2 decimals.",
    }),
    unit: msg({
      id: "metric.drawdown.unit",
      message: "Curve currency; cards display the decline as a negative amount.",
    }),
    caveat: msg({
      id: "metric.drawdown.caveat",
      message:
        "This is cash-ledger equity, not cumulative P&L alone or marked-to-market account value. Withdrawals can create drawdown. Opening balance is already a ledger deposit, never added twice. No positive drawdown displays — on these cards; report % mode scales the amount by its funding denominator.",
    }),
  },
  drawdownPct: {
    title: msg({ id: "metric.drawdownPct.title", message: "Equity drawdown percent" }),
    definition: msg({
      id: "metric.drawdownPct.definition",
      message: "Decline relative to the running equity peak within the selected chart window.",
    }),
    formula: msg({
      id: "metric.drawdownPct.formula",
      message:
        "For each visible point: (equity − running visible peak) ÷ peak × 100 when peak > 0, otherwise 0. Max is the most negative value; current is the last value.",
    }),
    unit: msg({
      id: "metric.drawdownPct.unit",
      message: "%, zero or negative; displayed to 2 decimals.",
    }),
    caveat: msg({
      id: "metric.drawdownPct.caveat",
      message:
        "The peak resets to the first visible point when the range changes. Cash flows affect this API equity curve; it excludes unrealized market value. Empty series gives 0 internally but the panel shows no data. This is not the summary amount divided by deposited capital.",
    }),
  },
  accountPerformance: {
    title: msg({ id: "metric.accountPerformance.title", message: "Account performance" }),
    definition: msg({
      id: "metric.accountPerformance.definition",
      message: "Net realized trading P&L for the current header filter scope.",
    }),
    formula: msg({
      id: "metric.accountPerformance.formula",
      message:
        "Sum of eligible closed-trade net P&L from the analytics summary, after fees. The active account and trade filters apply.",
    }),
    unit: msg({
      id: "metric.accountPerformance.unit",
      message: "Account/normalized display currency.",
    }),
    caveat: msg({
      id: "metric.accountPerformance.caveat",
      message:
        "Excludes open trades, including their partial-close P&L, and unrealized market value. Cash-ledger flows are not trading P&L. Current FX/display normalization is not historical FX attribution.",
    }),
  },
  accountReturn: {
    title: msg({ id: "metric.accountReturn.title", message: "Return percentage" }),
    definition: msg({
      id: "metric.accountReturn.definition",
      message:
        "Header net realized trading P&L relative to the selected accounts’ signed cash-ledger total.",
    }),
    formula: msg({
      id: "metric.accountReturn.formula",
      message:
        "Filtered closed-trade net P&L ÷ selected accounts’ signed cash-ledger total × 100. Opening balance is already a deposit and is not added twice.",
    }),
    unit: msg({
      id: "metric.accountReturn.unit",
      message: "%.",
    }),
    caveat: msg({
      id: "metric.accountReturn.caveat",
      message:
        "Hidden when funding ≤ 0 or required data/currency normalization is unavailable. Not time-weighted or money-weighted return: cash-flow dates are not weighted. Excludes open-trade and unrealized P&L. Current FX normalization is not historical FX attribution.",
    }),
  },
  accountRealizedPnl: {
    title: msg({ id: "metric.accountRealizedPnl.title", message: "Account realized P&L" }),
    definition: msg({
      id: "metric.accountRealizedPnl.definition",
      message: "Sum of available trade net P&L in this account’s loaded trade history.",
    }),
    formula: msg({
      id: "metric.accountRealizedPnl.formula",
      message:
        "Add each loaded trade’s net P&L after fees; missing P&L adds zero. This account’s history is used, independently of the header’s trade filters.",
    }),
    unit: msg({ id: "metric.accountRealizedPnl.unit", message: "Account base currency." }),
    caveat: msg({
      id: "metric.accountRealizedPnl.caveat",
      message:
        "Can include available partial-close realized P&L from open trades. Missing P&L is not an estimate. Excludes unrealized market value and cash-ledger flows.",
    }),
  },
  accountDetailReturn: {
    title: msg({ id: "metric.accountDetailReturn.title", message: "Account return percentage" }),
    definition: msg({
      id: "metric.accountDetailReturn.definition",
      message: "Account-detail realized P&L relative to this account’s signed cash-ledger total.",
    }),
    formula: msg({
      id: "metric.accountDetailReturn.formula",
      message:
        "Sum of available trade net P&L ÷ this account’s signed cash-ledger total × 100. Opening balance is already a deposit and is not added twice.",
    }),
    unit: msg({ id: "metric.accountDetailReturn.unit", message: "%." }),
    caveat: msg({
      id: "metric.accountDetailReturn.caveat",
      message:
        "Hidden only when the denominator is zero; a negative ledger total can reverse the percentage’s sign. Can include available partial-close P&L; missing P&L adds zero. Not time-weighted or money-weighted return, and excludes unrealized P&L.",
    }),
  },
  reportPnl: {
    title: msg({ id: "metric.reportPnl.title", message: "Reports P&L display" }),
    definition: msg({
      id: "metric.reportPnl.definition",
      message:
        "The report hero follows the net/gross and currency/% controls; underlying summary classifications remain net-based.",
    }),
    formula: msg({
      id: "metric.reportPnl.formula",
      message:
        "Net = summary net P&L. Gross hero = stored gross P&L, or net + fees for older APIs. % display = selected amount ÷ positive signed cash-ledger total × 100.",
    }),
    unit: msg({ id: "metric.reportPnl.unit", message: "Normalized display currency or %." }),
    caveat: msg({
      id: "metric.reportPnl.caveat",
      message:
        "The auxiliary Gross row is currently winning net amounts + absolute losing net amounts, not the before-fee hero. Fees % uses that same absolute-net base. Nonpositive funding disables % mode. No time/cash-flow weighting or unrealized P&L is added; filters and FX availability limit the sample.",
    }),
  },
  periodReturns: {
    title: msg({ id: "metric.periodReturns.title", message: "Period returns" }),
    definition: msg({
      id: "metric.periodReturns.definition",
      message: "Average recorded P&L per traded period, plus a linear annualized run rate.",
    }),
    formula: msg({
      id: "metric.periodReturns.formula",
      message:
        "Total selected net/gross P&L ÷ number of traded days, weeks or months. Annualized = total × 365 ÷ inclusive calendar-day span (minimum 1). % = result ÷ positive signed cash-ledger total × 100.",
    }),
    unit: msg({
      id: "metric.periodReturns.unit",
      message: "Display currency or %; weeks start Monday.",
    }),
    caveat: msg({
      id: "metric.periodReturns.caveat",
      message:
        "Uses closed trades with known net P&L, close dates on the configured display clock, and only traded periods in averages. No closed trades hides the strip. This is not compounded return, TWR or MWR; a short sample can produce an extreme annualized value.",
    }),
  },
  rollingWinRate: {
    title: msg({ id: "metric.rollingWinRate.title", message: "Rolling win rate" }),
    definition: msg({
      id: "metric.rollingWinRate.definition",
      message: "Win rate over the trailing selected number of chronological closed trades.",
    }),
    formula: msg({
      id: "metric.rollingWinRate.formula",
      message:
        "Positive net outcomes in the last N eligible trades ÷ N × 100; one point per trade after the window fills.",
    }),
    unit: msg({ id: "metric.rollingWinRate.unit", message: "%; N is the selected trade count." }),
    caveat: msg({
      id: "metric.rollingWinRate.caveat",
      message:
        "Breakeven occupies a denominator slot; open and unknown-net trades are excluded. Fewer than N trades gives no points. Net/gross display does not change this net-based series. The visible chart range can further restrict the input.",
    }),
  },
  evolution: {
    title: msg({ id: "metric.evolution.title", message: "Metric evolution" }),
    definition: msg({
      id: "metric.evolution.definition",
      message:
        "Cumulative-to-date trading metrics at each day/week/month bucket in the selected range.",
    }),
    formula: msg({
      id: "metric.evolution.formula",
      message:
        "Win rate = positive selected outcomes ÷ cumulative trade count. Profit factor = positive selected sum ÷ absolute negative selected sum (no losses: 0). P&L is the cumulative selected net/gross sum.",
    }),
    unit: msg({
      id: "metric.evolution.unit",
      message: "%; ratio; display currency or funding-based %.",
    }),
    caveat: msg({
      id: "metric.evolution.caveat",
      message:
        "Unlike summary win rate/profit factor, this chart classifies using the selected net/gross P&L. Breakeven stays in the win-rate denominator; unknown-net and open trades are excluded. Buckets use UTC dates, weeks start Monday. Cash flows and unrealized values are not added to cumulative P&L.",
    }),
  },
  simulationDrawdown: {
    title: msg({ id: "metric.simulationDrawdown.title", message: "Simulated maximum drawdown" }),
    definition: msg({
      id: "metric.simulationDrawdown.definition",
      message:
        "Median of the largest cumulative net-P&L decline in each bootstrap simulation path.",
    }),
    formula: msg({
      id: "metric.simulationDrawdown.formula",
      message:
        "Start each path at 0, resample historical closed-trade net P&L, track maximum (running peak − cumulative P&L), then take the 50th percentile across paths. The worse-case hint uses the 95th percentile.",
    }),
    unit: msg({
      id: "metric.simulationDrawdown.unit",
      message: "Display currency, or amount divided by the report funding denominator in % mode.",
    }),
    caveat: msg({
      id: "metric.simulationDrawdown.caveat",
      message:
        "This is simulated cumulative P&L, not account equity: cash flows and unrealized values are excluded. It always uses net outcomes, even in gross display mode. Simulation settings and sample size affect results; insufficient history shows no simulation.",
    }),
  },
} satisfies Record<string, MetricDefinition>;

export type MetricId = keyof typeof metricDefinitions;
