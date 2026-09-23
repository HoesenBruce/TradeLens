import { t as tr } from "@lingui/core/macro";
import {
  Calculator,
  CalendarDays,
  ChartLine,
  Globe,
  History,
  PartyPopper,
  RefreshCw,
  Scale,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

export type ToolId = "size" | "kelly" | "fx" | "today" | "chart" | "replay" | "econ" | "wrapped";

export type ToolGroupId = "calculators" | "markets" | "journal";

export interface ToolItem {
  id: ToolId;
  label: string;
  icon: LucideIcon;
  keywords: string[];
  group: ToolGroupId;
}

/** Section order for the tools menu — headings turn ten icons into three short lists. */
export const TOOL_GROUPS: { id: ToolGroupId; label: string }[] = [
  {
    id: "calculators",
    get label() {
      return tr({ id: "market.calculators", message: "Calculators" });
    },
  },
  {
    id: "markets",
    get label() {
      return tr({ id: "market.markets", message: "Markets" });
    },
  },
  {
    id: "journal",
    get label() {
      return tr({ id: "market.journal", message: "Journal" });
    },
  },
];

export const TOOL_ITEMS: ToolItem[] = [
  {
    id: "size",
    get label() {
      return tr({ id: "market.size", message: "Position size" });
    },
    icon: Calculator,
    keywords: ["calculator", "risk", "shares", "qty"],
    group: "calculators",
  },
  {
    id: "kelly",
    get label() {
      return tr({ id: "market.kelly", message: "Kelly criterion" });
    },
    icon: Scale,
    keywords: ["kelly", "sizing"],
    group: "calculators",
  },
  {
    id: "fx",
    get label() {
      return tr({ id: "market.fx", message: "Currency converter" });
    },
    icon: RefreshCw,
    keywords: ["forex", "fx", "currency"],
    group: "calculators",
  },
  {
    id: "today",
    get label() {
      return tr({ id: "market.today", message: "Today" });
    },
    icon: CalendarDays,
    keywords: ["calendar", "now"],
    group: "journal",
  },
  {
    id: "chart",
    get label() {
      return tr({ id: "market.advanced", message: "Advanced chart" });
    },
    icon: ChartLine,
    keywords: ["chart", "technical"],
    group: "markets",
  },
  {
    id: "replay",
    get label() {
      return tr({ id: "market.replay", message: "Replay" });
    },
    icon: History,
    keywords: ["backtest", "replay", "practice", "paper", "simulator"],
    group: "markets",
  },
  {
    id: "econ",
    get label() {
      return tr({ id: "market.econ", message: "Economic calendar" });
    },
    icon: Globe,
    keywords: ["news", "events", "macro"],
    group: "markets",
  },
  {
    id: "wrapped",
    get label() {
      return tr({ id: "market.wrapped", message: "Year Wrapped" });
    },
    icon: PartyPopper,
    keywords: ["recap", "annual", "year", "review"],
    group: "journal",
  },
];

export function toolsInGroup(group: ToolGroupId): ToolItem[] {
  return TOOL_ITEMS.filter((tool) => tool.group === group);
}
