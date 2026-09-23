import { tradeStatusLabel } from "@/lib/tradeLabels";
import { t as tr } from "@lingui/core/macro";
import type { ColumnDef, ColumnPinningState } from "@/lib/table";
import type { Trade } from "@/lib/api/types";
import { usePrivacyMode } from "@/lib/displayPrefs";
import { fmtDateTime, fmtDuration, fmtMoney, fmtSignedMoney, fmtTradeDay } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { resolveTradeDirection } from "@/lib/tradeDirection";
import { DirCell } from "./DirCell";
import { Pill, type PillTone } from "./Pill";
import { pnlColor } from "./theme-tokens";
import { TradeRowMenu, type TradeRowActions } from "./TradeRowMenu";

export type { TradeRowActions };

const MARKET_LABELS: Record<string, string> = {
  stock: "STK",
  option: "OPT",
  crypto: "CRY",
  futures: "FUT",
  forex: "FX",
};

function marketTitles(): Record<string, string> {
  return {
    stock: tr({ id: "trades.stock", message: "Stock" }),
    option: tr({ id: "trades.option", message: "Option" }),
    crypto: tr({ id: "trades.crypto", message: "Crypto" }),
    futures: tr({ id: "trades.futures", message: "Futures" }),
    forex: tr({ id: "trades.forex", message: "Forex" }),
  };
}

export function marketLabel(instrumentType: string): string {
  return MARKET_LABELS[instrumentType] ?? instrumentType.slice(0, 3).toUpperCase();
}

/** Conventional contract size when Trade list payloads omit fill multipliers. */
export function tradeNotionalMultiplier(instrumentType: string): number {
  return instrumentType === "option" ? 100 : 1;
}

export function tradeNotional(qty: number, price: number, instrumentType: string): number {
  return qty * price * tradeNotionalMultiplier(instrumentType);
}

export function tradeStatus(t: Trade): {
  label: "WIN" | "LOSS" | "OPEN" | "BE";
  tone: PillTone;
} {
  if (t.status === "open") return { label: "OPEN", tone: "accent" };
  if (t.net_pnl != null && t.net_pnl > 0) return { label: "WIN", tone: "pos" };
  if (t.net_pnl != null && t.net_pnl < 0) return { label: "LOSS", tone: "neg" };
  return { label: "BE", tone: "muted" };
}

/** Net P&L ÷ planned risk when journal initial_risk is set. */
export function tradeRMultiple(t: Trade): number | null {
  if (t.initial_risk == null || t.initial_risk <= 0 || t.net_pnl == null) return null;
  return t.net_pnl / t.initial_risk;
}

function muted(v: string, title?: string) {
  return (
    <span className="text-muted-foreground" title={title}>
      {v}
    </span>
  );
}

function MoneyCell({
  value,
  currency,
  fxRate = 1,
}: {
  value: number | null;
  currency: string;
  fxRate?: number;
}) {
  usePrivacyMode();
  if (value == null) return muted("-");
  const text = fmtMoney(value * fxRate, currency, intlLocale());
  return (
    <span className="tabular-nums" title={text}>
      {text}
    </span>
  );
}

function SignedMoneyCell({
  value,
  currency,
  fxRate = 1,
}: {
  value: number;
  currency: string;
  fxRate?: number;
}) {
  usePrivacyMode();
  const text = fmtSignedMoney(value * fxRate, currency, intlLocale());
  return (
    <span className={`tabular-nums font-semibold ${pnlColor(value)}`} title={text}>
      {text}
    </span>
  );
}

export function tradeColumns(
  currency: string,
  actions: TradeRowActions,
  fxRate = 1,
): ColumnDef<Trade>[] {
  return [
    {
      accessorKey: "symbol",
      header: tr({ id: "trades.symbol", message: "Symbol" }),
      meta: { label: tr({ id: "trades.symbol", message: "Symbol" }), minWidth: 72 },
      cell: (i) => (
        <span className="font-semibold text-primary" title={i.getValue<string>()}>
          {i.getValue<string>()}
        </span>
      ),
    },
    {
      accessorKey: "stock_name",
      header: tr({ id: "trades.stockName", message: "Stock Name" }),
      meta: { label: tr({ id: "trades.stockName", message: "Stock Name" }), minWidth: 120 },
      cell: (i) => i.getValue<string | undefined>() || muted("-"),
    },
    {
      id: "status",
      accessorFn: (row) => tradeStatus(row).label,
      header: tr({ id: "trades.status", message: "Status" }),
      meta: {
        label: tr({ id: "trades.status", message: "Status" }),
        headerTitle: tr({ id: "trades.tradeResult", message: "Trade result" }),
        minWidth: 64,
      },
      cell: (i) => {
        const s = tradeStatus(i.row.original);
        const titles: Record<typeof s.label, string> = {
          WIN: tr({ id: "trades.win", message: "Win" }),
          LOSS: tr({ id: "trades.loss", message: "Loss" }),
          OPEN: tr({ id: "trades.open", message: "Open" }),
          BE: tr({ id: "trades.breakEven", message: "Break-even" }),
        };
        return (
          <Pill tone={s.tone} title={titles[s.label]}>
            {tradeStatusLabel(s.label)}
          </Pill>
        );
      },
    },
    {
      id: "direction",
      accessorFn: (row) =>
        resolveTradeDirection({
          direction: row.direction,
          instrumentType: row.instrument_type,
          symbol: row.symbol,
        }).sortKey,
      header: tr({ id: "trades.direction", message: "Direction" }),
      meta: {
        label: tr({ id: "trades.direction", message: "Direction" }),
        headerTitle: tr({
          id: "trades.directionHint",
          message: "Direction — long/short; LC/LP/SC/SP when option",
        }),
        minWidth: 88,
      },
      cell: (i) => {
        const t = i.row.original;
        return (
          <DirCell direction={t.direction} instrumentType={t.instrument_type} symbol={t.symbol} />
        );
      },
    },
    {
      accessorKey: "instrument_type",
      header: tr({ id: "trades.market", message: "Market" }),
      meta: {
        label: tr({ id: "trades.market", message: "Market" }),
        headerTitle: tr({ id: "trades.instrumentType", message: "Instrument type" }),
      },
      cell: (i) => (
        <Pill tone="muted" title={marketTitles()[i.getValue<string>()]}>
          {marketLabel(i.getValue<string>())}
        </Pill>
      ),
    },
    {
      accessorKey: "qty_opened",
      header: tr({ id: "trades.quantity", message: "Quantity" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.quantity", message: "Quantity" }),
        headerTitle: tr({ id: "trades.quantityOpened", message: "Quantity opened" }),
        minWidth: 80,
      },
      cell: (i) => (
        <span className="tabular-nums" title={String(i.getValue<number>())}>
          {i.getValue<number>().toFixed(2)}
        </span>
      ),
    },
    {
      accessorKey: "avg_entry_price",
      header: tr({ id: "trades.entry", message: "Entry" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.entry", message: "Entry" }),
        headerTitle: tr({ id: "trades.avgEntry", message: "Average entry price" }),
        minWidth: 80,
      },
      cell: (i) => <MoneyCell value={i.getValue<number>()} currency={currency} fxRate={fxRate} />,
    },
    {
      accessorKey: "avg_exit_price",
      header: tr({ id: "trades.exit", message: "Exit" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.exit", message: "Exit" }),
        headerTitle: tr({ id: "trades.avgExit", message: "Average exit price" }),
        minWidth: 80,
      },
      cell: (i) => (
        <MoneyCell value={i.getValue<number | null>()} currency={currency} fxRate={fxRate} />
      ),
    },
    {
      id: "ent_tot",
      accessorFn: (row) => tradeNotional(row.qty_opened, row.avg_entry_price, row.instrument_type),
      header: tr({ id: "trades.entryTotal", message: "Entry total" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.entryTotal", message: "Entry total" }),
        headerTitle: tr({
          id: "trades.entryTotalHint",
          message: "Entry total — quantity × average entry × multiplier",
        }),
        minWidth: 96,
      },
      cell: (i) => {
        const t = i.row.original;
        return (
          <MoneyCell
            value={tradeNotional(t.qty_opened, t.avg_entry_price, t.instrument_type)}
            currency={currency}
            fxRate={fxRate}
          />
        );
      },
    },
    {
      id: "ext_tot",
      accessorFn: (row) =>
        row.avg_exit_price == null
          ? null
          : tradeNotional(row.qty_opened, row.avg_exit_price, row.instrument_type),
      header: tr({ id: "trades.exitTotal", message: "Exit total" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.exitTotal", message: "Exit total" }),
        headerTitle: tr({
          id: "trades.exitTotalHint",
          message: "Exit total — quantity × average exit × multiplier",
        }),
        minWidth: 96,
      },
      sortFn: (a, b, id) => {
        const av = a.getValue<number | null>(id);
        const bv = b.getValue<number | null>(id);
        if (av == null && bv == null) return 0;
        if (av == null) return 1;
        if (bv == null) return -1;
        return av === bv ? 0 : av < bv ? -1 : 1;
      },
      cell: (i) => {
        const t = i.row.original;
        return t.avg_exit_price == null ? (
          muted("-")
        ) : (
          <MoneyCell
            value={tradeNotional(t.qty_opened, t.avg_exit_price, t.instrument_type)}
            currency={currency}
            fxRate={fxRate}
          />
        );
      },
    },
    {
      id: "pos",
      accessorFn: (row) => {
        if (row.status !== "open") return null;
        return row.qty_remaining > 0 ? row.qty_remaining : row.qty_opened;
      },
      header: tr({ id: "trades.position", message: "Position" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.position", message: "Position" }),
        headerTitle: tr({ id: "trades.openPosition", message: "Position still open" }),
        minWidth: 80,
      },
      sortFn: (a, b, id) => {
        const av = a.getValue<number | null>(id);
        const bv = b.getValue<number | null>(id);
        if (av == null && bv == null) return 0;
        if (av == null) return 1;
        if (bv == null) return -1;
        return av === bv ? 0 : av < bv ? -1 : 1;
      },
      cell: (i) => {
        const t = i.row.original;
        if (t.status !== "open") return muted("-");
        const qty = t.qty_remaining > 0 ? t.qty_remaining : t.qty_opened;
        return (
          <span className="tabular-nums" title={String(qty)}>
            {qty.toFixed(2)}
          </span>
        );
      },
    },
    {
      accessorKey: "time_in_trade_secs",
      header: tr({ id: "trades.hold", message: "Hold" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.hold", message: "Hold" }),
        headerTitle: tr({ id: "trades.holdHint", message: "Time in trade" }),
        minWidth: 56,
      },
      cell: (i) => {
        const v = i.getValue<number | null>();
        return v == null || v <= 0 ? (
          muted("-")
        ) : (
          <span className="tabular-nums text-muted-foreground" title={fmtDuration(v)}>
            {fmtDuration(v)}
          </span>
        );
      },
    },
    {
      accessorKey: "fees_total",
      header: tr({ id: "trades.fees", message: "Fees" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.fees", message: "Fees" }),
        headerTitle: tr({ id: "trades.feesHint", message: "Total fees and commissions" }),
        minWidth: 72,
      },
      cell: (i) => <MoneyCell value={i.getValue<number>()} currency={currency} fxRate={fxRate} />,
    },
    {
      accessorKey: "net_pnl",
      header: tr({ id: "trades.pnl", message: "P&L" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.pnl", message: "P&L" }),
        headerTitle: tr({ id: "trades.netPnl", message: "Net P&L" }),
        minWidth: 112,
      },
      cell: (i) => {
        const v = i.getValue<number | null>();
        if (v == null) return muted("-");
        return <SignedMoneyCell value={v} currency={currency} fxRate={fxRate} />;
      },
    },
    {
      accessorKey: "return_pct",
      header: tr({ id: "trades.pnlPercent", message: "P&L %" }),
      meta: {
        align: "right",
        label: tr({ id: "trades.pnlPercent", message: "P&L %" }),
        headerTitle: tr({
          id: "trades.pnlPercentHint",
          message: "Net P&L as a percentage of entry total",
        }),
        minWidth: 88,
      },
      cell: (i) => {
        const v = i.getValue<number | null>();
        if (v == null) return muted("-");
        return (
          <span className={`tabular-nums ${pnlColor(v)}`} title={`${v.toFixed(2)}%`}>
            {v.toFixed(2)}%
          </span>
        );
      },
    },
    {
      accessorKey: "opened_at",
      header: tr({ id: "trades.date", message: "Date" }),
      meta: {
        label: tr({ id: "trades.createdAt", message: "Created At" }),
        headerTitle: tr({ id: "trades.openedDate", message: "Date opened" }),
        minWidth: 96,
      },
      cell: (i) => {
        const v = i.getValue<string>();
        return muted(fmtTradeDay(v), fmtDateTime(v));
      },
    },
    {
      accessorKey: "closed_at",
      header: tr({ id: "trades.closeColumn", message: "Close" }),
      meta: {
        label: tr({ id: "trades.closeDate", message: "Close date" }),
        headerTitle: tr({ id: "trades.closedDateHint", message: "Date closed (last activity)" }),
        minWidth: 96,
      },
      cell: (i) => {
        const v = i.getValue<string | null>();
        return v ? muted(fmtTradeDay(v), fmtDateTime(v)) : muted("-");
      },
    },
    {
      id: "r_multiple",
      accessorFn: (row) => tradeRMultiple(row),
      header: "R",
      meta: {
        align: "right",
        label: "R",
        headerTitle: tr({
          id: "trades.riskHint",
          message: "Net P&L ÷ planned risk (initial risk)",
        }),
        minWidth: 56,
      },
      cell: (i) => {
        const v = i.getValue<number | null>();
        if (v == null) return muted("-");
        const sign = v > 0 ? "+" : "";
        return (
          <span
            className={`tabular-nums font-semibold ${pnlColor(v)}`}
            title={`${sign}${v.toFixed(2)}R — ${tr({ id: "trades.riskHint", message: "Net P&L ÷ planned risk (initial risk)" })}`}
          >
            {sign}
            {v.toFixed(2)}R
          </span>
        );
      },
    },
    {
      id: "tags",
      accessorFn: (row) => row.tags.map((t) => t.name).join(", "),
      header: tr({ id: "trades.tags", message: "Tags" }),
      enableSorting: false,
      meta: {
        label: tr({ id: "trades.tags", message: "Tags" }),
        headerTitle: tr({ id: "trades.tradeTags", message: "Trade tags" }),
        minWidth: 96,
      },
      cell: (i) => {
        const tags = i.row.original.tags;
        if (tags.length === 0) return muted("-");
        const shown = tags.slice(0, 2);
        const rest = tags.length - shown.length;
        return (
          <div className="flex max-w-[14rem] items-center gap-1">
            {shown.map((t) => (
              <Pill
                key={t.id}
                tone={t.kind === "mistake" ? "neg" : "muted"}
                title={t.name}
                className="max-w-[7rem] overflow-hidden"
              >
                <span className="truncate">{t.name}</span>
              </Pill>
            ))}
            {rest > 0 ? (
              <span
                className="shrink-0 text-muted-foreground"
                title={tags
                  .slice(2)
                  .map((t) => t.name)
                  .join(", ")}
              >
                +{rest}
              </span>
            ) : null}
          </div>
        );
      },
    },
    {
      id: "actions",
      header: "",
      enableSorting: false,
      enableHiding: false,
      meta: { minWidth: 48 },
      cell: (i) => <TradeRowMenu trade={i.row.original} actions={actions} />,
    },
  ];
}

/** Pin the actions column via TanStack `columnPinning` (pass to DataTable). */
export const TRADE_COLUMN_PINNING: ColumnPinningState = { start: [], end: ["actions"] };

/** Sortable trade columns for the tablecn-style Sort button. */
export function tradeSortColumns(): { id: string; label: string }[] {
  return [
    { id: "opened_at", label: tr({ id: "trades.createdAt", message: "Created At" }) },
    { id: "closed_at", label: tr({ id: "trades.closeDate", message: "Close date" }) },
    { id: "symbol", label: tr({ id: "trades.symbol", message: "Symbol" }) },
    { id: "status", label: tr({ id: "trades.status", message: "Status" }) },
    { id: "direction", label: tr({ id: "trades.direction", message: "Direction" }) },
    { id: "instrument_type", label: tr({ id: "trades.market", message: "Market" }) },
    { id: "qty_opened", label: tr({ id: "trades.qty", message: "Qty" }) },
    { id: "avg_entry_price", label: tr({ id: "trades.entry", message: "Entry" }) },
    { id: "avg_exit_price", label: tr({ id: "trades.exit", message: "Exit" }) },
    { id: "ent_tot", label: tr({ id: "trades.entryTotal", message: "Entry total" }) },
    { id: "ext_tot", label: tr({ id: "trades.exitTotal", message: "Exit total" }) },
    { id: "pos", label: tr({ id: "trades.position", message: "Position" }) },
    { id: "time_in_trade_secs", label: tr({ id: "trades.hold", message: "Hold" }) },
    { id: "fees_total", label: tr({ id: "trades.fees", message: "Fees" }) },
    { id: "net_pnl", label: tr({ id: "trades.pnl", message: "P&L" }) },
    { id: "return_pct", label: tr({ id: "trades.pnlPercent", message: "P&L %" }) },
    { id: "r_multiple", label: "R" },
  ];
}

/** Hideable trade columns for the tablecn-style View button. */
export function tradeViewColumns(): { id: string; label: string }[] {
  return [
    { id: "symbol", label: tr({ id: "trades.symbol", message: "Symbol" }) },
    { id: "stock_name", label: tr({ id: "trades.stockName", message: "Stock Name" }) },
    { id: "status", label: tr({ id: "trades.status", message: "Status" }) },
    { id: "direction", label: tr({ id: "trades.direction", message: "Direction" }) },
    { id: "instrument_type", label: tr({ id: "trades.market", message: "Market" }) },
    { id: "qty_opened", label: tr({ id: "trades.qty", message: "Qty" }) },
    { id: "avg_entry_price", label: tr({ id: "trades.entry", message: "Entry" }) },
    { id: "avg_exit_price", label: tr({ id: "trades.exit", message: "Exit" }) },
    { id: "ent_tot", label: tr({ id: "trades.entryTotal", message: "Entry total" }) },
    { id: "ext_tot", label: tr({ id: "trades.exitTotal", message: "Exit total" }) },
    { id: "pos", label: tr({ id: "trades.position", message: "Position" }) },
    { id: "time_in_trade_secs", label: tr({ id: "trades.hold", message: "Hold" }) },
    { id: "fees_total", label: tr({ id: "trades.fees", message: "Fees" }) },
    { id: "net_pnl", label: tr({ id: "trades.pnl", message: "P&L" }) },
    { id: "return_pct", label: tr({ id: "trades.pnlPercent", message: "P&L %" }) },
    { id: "opened_at", label: tr({ id: "trades.createdAt", message: "Created At" }) },
    { id: "closed_at", label: tr({ id: "trades.closeDate", message: "Close date" }) },
    { id: "r_multiple", label: "R" },
    { id: "tags", label: tr({ id: "trades.tags", message: "Tags" }) },
  ];
}
