import { useLingui as useLinguiMacro } from "@lingui/react/macro";
import type { OnChangeFn, VisibilityState } from "@/lib/table";
import { Settings2 } from "lucide-react";
import { cn } from "@/lib/cn";
import { buttonVariants } from "./ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "./ui/menu";

export type ViewColumnOption = {
  id: string;
  label: string;
};

/**
 * tablecn-style View control — toggle column visibility.
 */
export function ViewOptions({
  columns,
  columnVisibility,
  onColumnVisibilityChange,
  className,
}: {
  columns: readonly ViewColumnOption[];
  columnVisibility: VisibilityState;
  onColumnVisibilityChange: OnChangeFn<VisibilityState>;
  className?: string;
}) {
  const { t: tr } = useLinguiMacro();

  function isVisible(id: string) {
    return columnVisibility[id] !== false;
  }

  function setVisible(id: string, visible: boolean) {
    onColumnVisibilityChange((prev) => ({
      ...prev,
      [id]: visible,
    }));
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          buttonVariants({ variant: "outline", size: "sm" }),
          "h-8 !bg-transparent hover:!bg-transparent aria-expanded:!bg-transparent",
          className,
        )}
        aria-label={tr({ id: "market.columns", message: "Toggle columns" })}
      >
        <Settings2 size={14} strokeWidth={1.75} />
        {tr({ id: "market.view", message: "View" })}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        <DropdownMenuGroup>
          <DropdownMenuLabel>
            {tr({ id: "market.columns", message: "Toggle columns" })}
          </DropdownMenuLabel>
          {columns.map((column) => (
            <DropdownMenuCheckboxItem
              key={column.id}
              checked={isVisible(column.id)}
              onCheckedChange={(checked) => setVisible(column.id, checked === true)}
            >
              {column.label}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
