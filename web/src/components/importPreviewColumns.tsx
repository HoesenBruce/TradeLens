import { t as tr } from "@lingui/core/macro";
import type { ColumnDef } from "@/lib/table";
import { Pencil } from "lucide-react";
import type { JournalTradePreview } from "@/lib/api/types";
import { cn } from "@/lib/cn";
import { usePrivacyMode } from "@/lib/displayPrefs";
import { fmtSignedMoney } from "@/lib/format";
import {
  effectiveOptionRight,
  formatMarketLabel,
  type OptionRightOverride,
} from "@/lib/importOptionRight";
import { intlLocale } from "@/lib/locale";
import { resolveTradeDirection } from "@/lib/tradeDirection";
import { DirCell } from "./DirCell";
import { Button } from "./ui/button";

function ReturnCell({ value, currency }: { value: number; currency: string }) {
  usePrivacyMode();
  const locale = intlLocale();
  return (
    <span
      className={cn("tabular-nums font-semibold", value >= 0 ? "text-profit" : "text-destructive")}
    >
      {fmtSignedMoney(value, currency, locale)}
    </span>
  );
}

export function journalTradePreviewColumns(
  currency: string,
  onEdit?: (trade: JournalTradePreview) => void,
  optionOverrides: Record<number, OptionRightOverride> = {},
): ColumnDef<JournalTradePreview>[] {
  const columns: ColumnDef<JournalTradePreview>[] = [
    {
      accessorKey: "symbol",
      header: tr({ id: "imports.symbol", message: "Symbol" }),
      cell: (info) => (
        <span className="font-medium text-foreground">{info.getValue<string>()}</span>
      ),
    },
    {
      id: "market",
      header: tr({ id: "imports.market", message: "Market" }),
      accessorFn: (row) => formatMarketLabel(row),
      cell: (info) => (
        <span className="text-[11px] uppercase tracking-wide text-muted-foreground">
          {formatMarketLabel(info.row.original)}
        </span>
      ),
    },
    {
      id: "direction",
      header: tr({ id: "imports.dir", message: "Dir" }),
      accessorFn: (row) =>
        resolveTradeDirection({
          direction: row.side,
          instrumentType: row.instrument_type,
          optionRight: effectiveOptionRight(row, optionOverrides),
          symbol: row.symbol,
          markMissingOptionRight: true,
        }).sortKey,
      meta: {
        label: tr({ id: "imports.direction", message: "Direction" }),
        headerTitle: tr({
          id: "imports.directionHint",
          message: "Direction — long/short, call/put when option",
        }),
      },
      cell: (info) => {
        const trade = info.row.original;
        return (
          <DirCell
            direction={trade.side}
            instrumentType={trade.instrument_type}
            optionRight={effectiveOptionRight(trade, optionOverrides)}
            symbol={trade.symbol}
            markMissingOptionRight
          />
        );
      },
    },
    {
      accessorKey: "qty",
      header: tr({ id: "imports.qty", message: "Qty" }),
      meta: { align: "right" },
      cell: (info) => <span className="tabular-nums">{info.getValue<number>()}</span>,
    },
    {
      accessorKey: "entry",
      header: tr({ id: "imports.entry", message: "Entry" }),
      meta: { align: "right" },
      cell: (info) => <span className="tabular-nums">{info.getValue<number>().toFixed(2)}</span>,
    },
    {
      accessorKey: "exit",
      header: tr({ id: "imports.exit", message: "Exit" }),
      meta: { align: "right" },
      cell: (info) => <span className="tabular-nums">{info.getValue<number>().toFixed(2)}</span>,
    },
    {
      accessorKey: "return_usd",
      header: tr({ id: "imports.pnl", message: "P&L" }),
      meta: { align: "right" },
      cell: (info) => <ReturnCell value={info.getValue<number>()} currency={currency} />,
    },
  ];

  if (onEdit) {
    columns.push({
      id: "actions",
      header: "",
      enableSorting: false,
      meta: { align: "right" },
      cell: (info) => (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={tr({ id: "imports.editSymbol", message: `Edit ${info.row.original.symbol}` })}
          title={tr({ id: "imports.editTrade", message: "Edit trade" })}
          onClick={(e) => {
            e.stopPropagation();
            onEdit(info.row.original);
          }}
        >
          <Pencil size={12} strokeWidth={1.75} />
        </Button>
      ),
    });
  }

  return columns;
}

export function csvSampleColumns(headers: string[]): ColumnDef<Record<string, string>>[] {
  return headers.map((header) => ({
    id: header,
    accessorFn: (row) => row[header] ?? "",
    header,
    enableSorting: false,
    cell: (info) => (
      <span className="text-[11px] tabular-nums text-muted-foreground">
        {String(info.getValue() || "-")}
      </span>
    ),
  }));
}
