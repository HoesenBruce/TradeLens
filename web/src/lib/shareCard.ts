import { t as localize } from "@lingui/core/macro";
import type { TradeDetail } from "@/lib/api/types";
import { getDisplayTimeOpts } from "@/lib/displayPrefs";
import { fmtPct, fmtSignedMoney } from "@/lib/format";
import type { TradeInsights } from "@/lib/tradeInsights";
import type { YearWrapped } from "@/lib/wrapped";

export interface ShareCardStat {
  label: string;
  value: string;
}

export type ShareCardTone = "profit" | "loss" | "flat";
export type ShareCardChipTone = ShareCardTone | "muted" | "brand";

export interface ShareCardChip {
  text: string;
  tone: ShareCardChipTone;
}

export interface ShareCardData {
  title: string;
  /** Accessible name for the rendered SVG. */
  ariaLabel: string;
  chips: ShareCardChip[];
  tone: ShareCardTone;
  dateLabel: string;
  hero: ShareCardStat;
  stats: ShareCardStat[];
}

function signedPct(v: number): string {
  return `${v >= 0 ? "+" : ""}${v.toFixed(2)}%`;
}

function signedR(v: number): string {
  return `${v >= 0 ? "+" : ""}${v.toFixed(2)}R`;
}

/**
 * Build the data for a shareable trade card. Privacy-first: dollar amounts
 * appear only when showAmounts is explicitly on — the default card speaks in
 * R multiples and percentages, which brag without disclosing account size.
 */
export function buildTradeShareCard(
  trade: TradeDetail,
  insights: TradeInsights,
  opts: {
    showAmounts: boolean;
    locale: string;
    /** Privacy-bound formatter from useMoneyFormatters(); defaults to the module one. */
    fmtSignedMoney?: typeof fmtSignedMoney;
  },
): ShareCardData {
  const net = trade.net_pnl ?? 0;
  const closed = trade.status === "closed";
  const outcome = !closed ? "OPEN" : net > 0 ? "WIN" : net < 0 ? "LOSS" : "FLAT";
  const outcomeLabel = {
    get OPEN() {
      return localize({ id: "wrapped.outcomeOpen", message: "OPEN" });
    },
    get WIN() {
      return localize({ id: "wrapped.outcomeWin", message: "WIN" });
    },
    get LOSS() {
      return localize({ id: "wrapped.outcomeLoss", message: "LOSS" });
    },
    get FLAT() {
      return localize({ id: "wrapped.outcomeFlat", message: "FLAT" });
    },
  }[outcome];
  const tone = closed && net > 0 ? "profit" : closed && net < 0 ? "loss" : "flat";

  const when = trade.closed_at ?? trade.opened_at;
  // Same display timezone as the rest of the app, so the card date matches
  // the trade header rather than the viewer's OS clock.
  const { timeZone } = getDisplayTimeOpts();
  const dateLabel = new Date(when).toLocaleDateString(opts.locale, {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone,
  });

  const r = insights.rMultiple;
  const pct = insights.returnPct;

  let hero: ShareCardStat;
  if (opts.showAmounts) {
    hero = {
      label: localize({ id: "accounts.netPnl", message: "Net P&L" }),
      value: (opts.fmtSignedMoney ?? fmtSignedMoney)(net, trade.pnl_currency, opts.locale),
    };
  } else if (r != null) {
    hero = {
      label: localize({ id: "wrapped.rMultiple", message: "R multiple" }),
      value: signedR(r),
    };
  } else if (pct != null) {
    hero = { label: localize({ id: "wrapped.return", message: "Return" }), value: signedPct(pct) };
  } else {
    hero = {
      label: localize({ id: "trades.status", message: "Status" }),
      value:
        outcome === "OPEN"
          ? localize({ id: "trades.open", message: "Open" })
          : outcomeLabel.toLowerCase(),
    };
  }

  const candidates: (ShareCardStat | null)[] = [
    opts.showAmounts && r != null
      ? { label: localize({ id: "wrapped.rMultiple", message: "R multiple" }), value: signedR(r) }
      : null,
    pct != null && (opts.showAmounts || r != null)
      ? { label: localize({ id: "wrapped.return", message: "Return" }), value: signedPct(pct) }
      : null,
    insights.holdLabel
      ? { label: localize({ id: "trades.hold", message: "Hold" }), value: insights.holdLabel }
      : null,
    insights.setupName
      ? { label: localize({ id: "accounts.setup", message: "Setup" }), value: insights.setupName }
      : null,
  ];
  const stats = candidates.filter((s): s is ShareCardStat => s != null).slice(0, 3);

  return {
    title: trade.symbol,
    ariaLabel: localize({
      id: "wrapped.value0TradeCard",
      message: `${{ value0: trade.symbol }} trade card`,
    }),
    chips: [
      {
        text:
          trade.direction === "short"
            ? localize({ id: "trades.shortUpper", message: "SHORT" })
            : localize({ id: "trades.longUpper", message: "LONG" }),
        tone: "muted",
      },
      { text: outcomeLabel, tone: outcome === "OPEN" ? "brand" : tone },
    ],
    tone,
    dateLabel,
    hero,
    stats,
  };
}

/**
 * Build the data for a shareable Year Wrapped card. Same privacy stance as
 * the trade card: the default speaks in win rate and ratios — dollars appear
 * only when showAmounts is explicitly on.
 */
export function buildWrappedShareCard(
  wrapped: YearWrapped,
  opts: {
    showAmounts: boolean;
    locale: string;
    currency: string;
    fxRate: number;
    /** True while the year is still running (current year → "Year to date"). */
    inProgress?: boolean;
    /** Privacy-bound formatter from useMoneyFormatters(); defaults to the module one. */
    fmtSignedMoney?: typeof fmtSignedMoney;
  },
): ShareCardData {
  const tone: ShareCardTone = wrapped.netPnl > 0 ? "profit" : wrapped.netPnl < 0 ? "loss" : "flat";
  const money = (v: number) =>
    (opts.fmtSignedMoney ?? fmtSignedMoney)(v * opts.fxRate, opts.currency, opts.locale);
  const winRate = fmtPct(wrapped.winRate, opts.locale);
  const profitFactor = wrapped.profitFactor > 0 ? wrapped.profitFactor.toFixed(2) : "0.00";

  const hero: ShareCardStat = opts.showAmounts
    ? {
        label: localize({ id: "accounts.netPnl", message: "Net P&L" }),
        value: money(wrapped.netPnl),
      }
    : { label: localize({ id: "accounts.winRate", message: "Win rate" }), value: winRate };

  const candidates: (ShareCardStat | null)[] = [
    opts.showAmounts
      ? { label: localize({ id: "accounts.winRate", message: "Win rate" }), value: winRate }
      : null,
    {
      label: localize({ id: "accounts.profitFactor", message: "Profit factor" }),
      value: profitFactor,
    },
    opts.showAmounts && wrapped.bestDay
      ? {
          label: localize({ id: "accounts.bestDay", message: "Best day" }),
          value: money(wrapped.bestDay.pnl),
        }
      : null,
    !opts.showAmounts
      ? {
          label: localize({ id: "wrapped.greenDays", message: "Green days" }),
          value: localize({
            id: "wrapped.value0OfValue1",
            message: `${{ value0: wrapped.greenDays }} of ${{ value1: wrapped.tradingDays }}`,
          }),
        }
      : null,
    !opts.showAmounts && wrapped.bestStreak > 0
      ? {
          label: localize({ id: "accounts.bestStreak", message: "Best streak" }),
          value: localize({
            id: "wrapped.value0Wins",
            message: `${{ value0: wrapped.bestStreak }} wins`,
          }),
        }
      : null,
  ];

  return {
    title: localize({
      id: "wrapped.value0Wrapped",
      message: `${{ value0: wrapped.year }} Wrapped`,
    }),
    ariaLabel: localize({
      id: "wrapped.value0WrappedShareCard",
      message: `${{ value0: wrapped.year }} Wrapped share card`,
    }),
    chips: [
      {
        text: localize({
          id: "wrapped.value0Trades",
          message: `${{ value0: wrapped.totalTrades }} TRADES`,
        }),
        tone: "muted",
      },
      {
        text:
          tone === "profit"
            ? localize({ id: "wrapped.greenYear", message: "GREEN YEAR" })
            : tone === "loss"
              ? localize({ id: "wrapped.redYear", message: "RED YEAR" })
              : localize({ id: "wrapped.flatYear", message: "FLAT YEAR" }),
        tone,
      },
    ],
    tone,
    dateLabel: opts.inProgress
      ? localize({ id: "wrapped.yearToDate", message: "Year to date" })
      : localize({ id: "wrapped.fullYear", message: "Full year" }),
    hero,
    stats: candidates.filter((s): s is ShareCardStat => s != null).slice(0, 3),
  };
}
