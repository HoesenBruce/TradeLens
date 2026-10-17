import type { Trade } from "./api/types";
import { finalTradeResult } from "./tradeFilters";

type CSVValue = string | number | boolean | null | undefined;

function csvCell(value: CSVValue): string {
  let text = value == null ? "" : String(value);
  // Spreadsheet applications must not execute symbols or other user-controlled text.
  if (typeof value === "string" && /^[\s]*[=+\-@]/.test(text)) text = `'${text}`;
  return `"${text.replaceAll('"', '""')}"`;
}

/** Export the exact matching collection, before UI pagination. Null values stay empty. */
export function tradesCSV(trades: Trade[]): string {
  const rows: CSVValue[][] = [
    [
      "Trade ID",
      "Symbol",
      "Instrument type",
      "Direction",
      "Status",
      "Trade result",
      "Position type",
      "Opened at",
      "Closed at",
      "Entry quantity",
      "Closed quantity",
      "Remaining quantity",
      "Close progress",
      "Holding precision",
      "Elapsed seconds",
      "Calendar days",
      "Market time zone",
      "Ongoing",
      "Net P&L",
      "P&L currency",
      "Tags",
    ],
  ];
  for (const t of trades)
    rows.push([
      t.id,
      t.symbol,
      t.instrument_type,
      t.direction,
      t.status,
      finalTradeResult(t),
      t.position_type,
      t.opened_at,
      t.closed_at,
      t.quantities?.entry_qty,
      t.quantities?.exit_qty,
      t.quantities?.open_qty,
      t.quantities?.lifecycle ?? "unknown",
      t.holding?.kind ?? "unknown",
      t.holding?.seconds,
      t.holding?.calendar_days,
      t.holding?.time_zone,
      t.holding?.ongoing,
      t.net_pnl,
      t.pnl_currency,
      t.tags.map((tag) => tag.name).join("; "),
    ]);
  return "\uFEFF" + rows.map((row) => row.map(csvCell).join(",")).join("\r\n") + "\r\n";
}
