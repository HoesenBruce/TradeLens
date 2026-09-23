import { useLingui as useLinguiMacro } from "@lingui/react/macro";
import { CircleUser, LogOut } from "lucide-react";
import { useState } from "react";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { useFilters } from "@/lib/filters";
import { useAccounts } from "@/lib/hooks/useAccounts";
import { RailTooltip } from "./RailTooltip";
import {
  Menu,
  MenuCheckboxItem,
  MenuGroup,
  MenuGroupLabel,
  MenuItem,
  MenuPopup,
  MenuSeparator,
  MenuTrigger,
} from "./ui/menu";

function normCurrency(c?: string): string {
  return c?.trim().toUpperCase() ?? "";
}

/** Menu row: account name, then its ledger currency (or the account count) as a quiet hint. */
function AccountMenuItem({
  checked,
  disabled,
  onCheckedChange,
  closeOnClick,
  label,
  hint,
}: {
  checked: boolean;
  disabled?: boolean;
  onCheckedChange: () => void;
  closeOnClick?: boolean;
  label: string;
  hint?: string;
}) {
  return (
    <MenuCheckboxItem
      checked={checked}
      disabled={disabled}
      closeOnClick={closeOnClick ?? false}
      onCheckedChange={onCheckedChange}
      className="pe-2.5"
    >
      <span className="flex w-full items-center gap-2">
        <span className="min-w-0 flex-1 truncate">{label}</span>
        {hint ? <span className="shrink-0 text-[11px] text-muted-foreground">{hint}</span> : null}
      </span>
    </MenuCheckboxItem>
  );
}

export function AccountNavPopover({ variant = "rail" }: { variant?: "rail" | "header" }) {
  const { t: tr } = useLinguiMacro();

  const [open, setOpen] = useState(false);
  const signOut = useAuth((s) => s.signOut);
  const { data: accounts, isLoading } = useAccounts();
  const accountIds = useFilters((s) => s.accountIds);
  const setAccounts = useFilters((s) => s.setAccounts);

  const items = accounts ?? [];
  const selected = accountIds ?? [];
  const selectedLabel = selected.length
    ? selected.length === 1
      ? (items.find((a) => a.id === selected[0])?.name ??
        tr({ id: "accounts.account", message: "Account" }))
      : tr({ id: "accounts.selectedCount", message: `${selected.length} accounts` })
    : tr({ id: "accounts.all", message: "All accounts" });
  const filterActive = selected.length > 0;
  const isRail = variant === "rail";
  const allAccountsHint = items.length
    ? tr({ id: "accounts.accountCount", message: `${items.length} accounts` })
    : undefined;

  // Portfolio selections must share one base currency — mixed sums are
  // meaningless and the API rejects them — so incompatible rows are disabled.
  const scopeCurrency = selected.length
    ? normCurrency(items.find((a) => a.id === selected[0])?.base_currency)
    : "";

  const toggleAccount = (id: string) => {
    const next = selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id];
    setAccounts(next);
  };

  return (
    <Menu open={open} onOpenChange={setOpen}>
      <MenuTrigger
        title={selectedLabel}
        aria-label={tr({ id: "accounts.accountScope", message: `Account: ${selectedLabel}` })}
        className={cn(
          "group relative flex cursor-pointer items-center justify-center rounded-md outline-none",
          isRail ? "size-9" : "size-8",
          "pointer-coarse:size-11",
          "transition-[background-color,color,transform] duration-150 ease-out",
          "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
          "motion-reduce:transition-none",
          open || filterActive
            ? "bg-accent text-foreground"
            : "text-muted-foreground hover:bg-accent hover:text-foreground",
        )}
      >
        <CircleUser
          size={15}
          strokeWidth={1.75}
          className="transition-transform duration-150 ease-out group-hover:scale-105 motion-reduce:transition-none"
        />
        {isRail ? <RailTooltip label={selectedLabel} /> : null}
      </MenuTrigger>
      <MenuPopup
        side={isRail ? "right" : "bottom"}
        align="end"
        sideOffset={isRail ? 8 : 6}
        className="w-56"
      >
        {isLoading ? (
          <p className="m-0 px-2 py-1.5 text-sm text-muted-foreground">
            {tr({ id: "accounts.loading", message: "Loading…" })}
          </p>
        ) : (
          <MenuGroup>
            <MenuGroupLabel>{tr({ id: "accounts.account", message: "Account" })}</MenuGroupLabel>
            <AccountMenuItem
              checked={!filterActive}
              closeOnClick
              onCheckedChange={() => setAccounts(undefined)}
              label={tr({ id: "accounts.all", message: "All accounts" })}
              hint={allAccountsHint}
            />
            {items.map((account) => {
              const currency = normCurrency(account.base_currency);
              const isSelected = selected.includes(account.id);
              const incompatible =
                Boolean(scopeCurrency) && !isSelected && currency !== scopeCurrency;
              return (
                <AccountMenuItem
                  key={account.id}
                  checked={isSelected}
                  disabled={incompatible}
                  onCheckedChange={() => toggleAccount(account.id)}
                  label={account.name}
                  hint={currency || undefined}
                />
              );
            })}
          </MenuGroup>
        )}
        {/* Hairline, not spacing: without it the action reads as another account row. */}
        <MenuSeparator />
        <MenuItem onClick={signOut}>
          <LogOut aria-hidden />
          {tr({ id: "accounts.signOut", message: "Sign out" })}
        </MenuItem>
      </MenuPopup>
    </Menu>
  );
}
