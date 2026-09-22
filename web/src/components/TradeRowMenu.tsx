import { useLingui } from "@lingui/react";
import { t as tr } from "@lingui/core/macro";
import { Copy, ExternalLink, Filter, MoreVertical, PanelRight, Trash2 } from "lucide-react";
import { useId, useState } from "react";
import type { Trade } from "@/lib/api/types";
import { cn } from "@/lib/cn";
import { useDeleteTrade } from "@/lib/hooks/useTradeDetail";
import { Modal } from "./Modal";
import { FormInput } from "./FormInput";
import { useToastManager } from "./Toast";
import { Button } from "./ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./ui/menu";

export interface TradeRowActions {
  /** Quick peek drawer. Omit when the list already opens the full page on row click. */
  onOpenDrawer?: (trade: Trade) => void;
  /** Navigate to /trades/:id */
  onOpenFullPage: (trade: Trade) => void;
  /** Set the global symbol filter */
  onFilterSymbol?: (symbol: string) => void;
  /** Called after a successful delete (e.g. close an open peek drawer). */
  onDeleted?: (trade: Trade) => void;
}

const triggerClass = cn(
  "-my-2 flex h-9 w-9 cursor-pointer items-center justify-center rounded-md",
  "text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
  "outline-none focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-ring",
);

export function TradeRowMenu({ trade, actions }: { trade: Trade; actions: TradeRowActions }) {
  useLingui();
  const [menuOpen, setMenuOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [typedConfirm, setTypedConfirm] = useState("");
  const confirmInputId = useId();
  const toast = useToastManager();
  const deleteTrade = useDeleteTrade();

  const canDelete = typedConfirm.trim().toUpperCase() === trade.symbol.trim().toUpperCase();

  async function copySymbol() {
    try {
      await navigator.clipboard.writeText(trade.symbol);
      toast.add({
        title: tr({ id: "trades.copied", message: "Copied" }),
        description: trade.symbol,
      });
    } catch {
      toast.add({
        title: tr({ id: "trades.copyFailed", message: "Could not copy" }),
        description: tr({ id: "trades.clipboardBlocked", message: "Clipboard access was blocked" }),
      });
    }
  }

  function closeDeleteModal(next: boolean) {
    setDeleteOpen(next);
    if (!next) setTypedConfirm("");
  }

  async function handleDelete() {
    try {
      await deleteTrade.mutateAsync(trade.id);
      toast.add({
        title: tr({ id: "trades.removed", message: "Trade removed" }),
        description: trade.symbol,
      });
      closeDeleteModal(false);
      actions.onDeleted?.(trade);
    } catch {
      toast.add({
        title: tr({ id: "trades.removeFailed", message: "Could not remove trade" }),
        description: tr({ id: "trades.retryLater", message: "Try again in a moment" }),
      });
    }
  }

  return (
    <div
      className="contents"
      onClick={(e) => e.stopPropagation()}
      onPointerDown={(e) => e.stopPropagation()}
    >
      <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
        <DropdownMenuTrigger
          aria-label={tr({ id: "trades.actionsForSymbol", message: `Actions for ${trade.symbol}` })}
          className={cn(triggerClass, menuOpen && "bg-accent text-foreground")}
        >
          <MoreVertical size={14} strokeWidth={1.5} aria-hidden />
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          side="bottom"
          sideOffset={4}
          className="min-w-[11.5rem] p-1"
        >
          {actions.onOpenDrawer ? (
            <DropdownMenuItem
              onClick={() => {
                actions.onOpenDrawer?.(trade);
              }}
            >
              <PanelRight size={14} strokeWidth={1.5} aria-hidden />
              {tr({ id: "trades.openDrawer", message: "Open drawer" })}
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuItem
            onClick={() => {
              actions.onOpenFullPage(trade);
            }}
          >
            <ExternalLink size={14} strokeWidth={1.5} aria-hidden />
            {tr({ id: "trades.openFull", message: "Open full page" })}
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => {
              void copySymbol();
            }}
          >
            <Copy size={14} strokeWidth={1.5} aria-hidden />
            {tr({ id: "trades.copySymbol", message: "Copy symbol" })}
          </DropdownMenuItem>
          {actions.onFilterSymbol ? (
            <DropdownMenuItem
              onClick={() => {
                actions.onFilterSymbol?.(trade.symbol);
              }}
            >
              <Filter size={14} strokeWidth={1.5} aria-hidden />
              {tr({ id: "trades.filterSymbol", message: `Filter by ${trade.symbol}` })}
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuSeparator />
          <DropdownMenuItem
            variant="destructive"
            onClick={() => {
              setMenuOpen(false);
              setDeleteOpen(true);
            }}
          >
            <Trash2 size={14} strokeWidth={1.5} aria-hidden />
            {tr({ id: "trades.remove", message: "Remove" })}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Modal
        open={deleteOpen}
        onOpenChange={closeDeleteModal}
        title={tr({ id: "trades.removeSymbolConfirm", message: `Remove ${trade.symbol}?` })}
        className="max-w-[min(336px,94vw)]"
        footer={
          <>
            <Button
              type="button"
              variant="ghost"
              disabled={deleteTrade.isPending}
              onClick={() => closeDeleteModal(false)}
            >
              {tr({ id: "trades.cancel", message: "Cancel" })}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={!canDelete || deleteTrade.isPending}
              onClick={() => void handleDelete()}
              className="border-transparent bg-destructive/15 hover:bg-destructive/25"
            >
              {deleteTrade.isPending
                ? tr({ id: "trades.removing", message: "Removing…" })
                : tr({ id: "trades.removeTrade", message: "Remove trade" })}
            </Button>
          </>
        }
      >
        <p className="m-0 text-[13px] leading-relaxed text-muted-foreground">
          {tr({
            id: "trades.removeWarning",
            message: "Permanently deletes this trade and all of its fills. This cannot be undone.",
          })}
        </p>
        <div>
          <label
            htmlFor={confirmInputId}
            className="mb-1.5 block text-[11px] text-muted-foreground"
          >
            {tr({ id: "trades.confirmSymbol", message: `Type ${trade.symbol} to confirm` })}
          </label>
          <FormInput
            id={confirmInputId}
            value={typedConfirm}
            onChange={(e) => setTypedConfirm(e.target.value)}
            autoFocus
            autoComplete="off"
            spellCheck={false}
            aria-label={tr({
              id: "trades.typeSymbolConfirm",
              message: `Type ${trade.symbol} to confirm`,
            })}
          />
        </div>
      </Modal>
    </div>
  );
}
