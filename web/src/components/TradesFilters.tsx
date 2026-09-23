import { useLingui } from "@lingui/react/macro";
import { ChartCandlestick, CircleDot, ListFilterPlus, Search, Tags } from "lucide-react";
import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import {
  Filters,
  type Filter,
  type FilterFieldConfig,
  type FilterOperator,
} from "@/components/filters";
import type {
  MarketFacetOption,
  SymbolFacetOption,
  TagFacetOption,
  TradeStatusFilter,
} from "@/lib/tradeFilters";

type TradesFilterValue = string;

function stableFilter(
  field: string,
  operator: string,
  values: TradesFilterValue[],
): Filter<TradesFilterValue> {
  return { id: field, field, operator, values };
}

function valuesOf(filters: Filter<TradesFilterValue>[], field: string): string[] {
  const match = filters.find((f) => f.field === field);
  if (!match?.values?.length) return [];
  return match.values.map(String);
}

function operatorOf(filters: Filter<TradesFilterValue>[], field: string, fallback: string): string {
  return filters.find((f) => f.field === field)?.operator ?? fallback;
}

export function TradesFilters({
  triggerClassName,
  iconOnly = false,
  symbols,
  onSymbolsChange,
  symbolOptions = [],
  tradeStatus,
  onToggleTradeStatus,
  onClearStatus,
  markets,
  marketOptions,
  onMarketsChange,
  tagIds,
  tagOptions,
  onTagIdsChange,
}: {
  /** Styling for the "Add filter" trigger so it matches the sibling toolbar controls. */
  triggerClassName?: string;
  /** Drop the trigger label to the accessible name only — compact toolbars. */
  iconOnly?: boolean;
  symbols: string[];
  onSymbolsChange: (symbols?: string[]) => void;
  symbolOptions?: SymbolFacetOption[];
  tradeStatus?: TradeStatusFilter;
  onToggleTradeStatus?: (filter: TradeStatusFilter) => void;
  onClearStatus?: () => void;
  markets?: string[];
  marketOptions?: MarketFacetOption[];
  onMarketsChange?: (markets?: string[]) => void;
  tagIds?: string[];
  tagOptions?: TagFacetOption[];
  onTagIdsChange?: (ids?: string[]) => void;
}) {
  // Persist operators so choosing "is not any of" / "includes all" doesn't snap back.
  // Values live in the parent store; operators are UI-only until the data layer grows modes.
  const { t: tr } = useLingui();
  const [ops, setOps] = useState<Record<string, string>>({});

  const fields = useMemo(() => {
    const STATUS_OPTIONS = [
      { value: "win", label: tr({ id: "trades.wins", message: "Wins" }) },
      { value: "loss", label: tr({ id: "trades.losses", message: "Losses" }) },
      { value: "open", label: tr({ id: "trades.open", message: "Open" }) },
      { value: "wash", label: tr({ id: "trades.wash", message: "Wash" }) },
    ] as const;

    /** Scalar facets (one value per trade) — only OR / NOT OR make sense. */
    const SCALAR_MULTI_OPS: FilterOperator[] = [
      { value: "is_any_of", label: tr({ id: "trades.anyOf", message: "is any of" }) },
      { value: "is_not_any_of", label: tr({ id: "trades.notAnyOf", message: "is not any of" }) },
    ];

    /** Tag facets (many per trade) — OR, NOT OR, and AND. */
    const TAG_MULTI_OPS: FilterOperator[] = [
      { value: "is_any_of", label: tr({ id: "trades.anyOf", message: "is any of" }) },
      { value: "is_not_any_of", label: tr({ id: "trades.notAnyOf", message: "is not any of" }) },
      { value: "includes_all", label: tr({ id: "trades.includesAll", message: "includes all" }) },
    ];

    const STATUS_OPS: FilterOperator[] = [
      { value: "is", label: tr({ id: "trades.is", message: "is" }) },
      { value: "is_not", label: tr({ id: "trades.isNot", message: "is not" }) },
    ];

    const next: FilterFieldConfig<TradesFilterValue>[] = [
      {
        key: "symbol",
        label: tr({ id: "trades.symbol", message: "Symbol" }),
        type: "multiselect",
        icon: <Search className="size-3.5" strokeWidth={1.75} />,
        searchable: true,
        operators: SCALAR_MULTI_OPS,
        options: symbolOptions.map((o) => ({
          value: o.value,
          label: o.label,
        })),
      },
    ];

    if (onToggleTradeStatus) {
      next.push({
        key: "status",
        label: tr({ id: "trades.status", message: "Status" }),
        type: "select",
        icon: <CircleDot className="size-3.5" strokeWidth={1.75} />,
        searchable: false,
        operators: STATUS_OPS,
        options: STATUS_OPTIONS.map((o) => ({ value: o.value, label: o.label })),
      });
    }

    if (onMarketsChange && marketOptions && marketOptions.length > 0) {
      next.push({
        key: "market",
        label: tr({ id: "trades.market", message: "Market" }),
        type: "multiselect",
        icon: <ChartCandlestick className="size-3.5" strokeWidth={1.75} />,
        searchable: true,
        operators: SCALAR_MULTI_OPS,
        options: marketOptions.map((o) => ({ value: o.value, label: o.label })),
      });
    }

    if (onTagIdsChange && tagOptions && tagOptions.length > 0) {
      next.push({
        key: "tags",
        label: tr({ id: "trades.tags", message: "Tags" }),
        type: "multiselect",
        icon: <Tags className="size-3.5" strokeWidth={1.75} />,
        searchable: true,
        operators: TAG_MULTI_OPS,
        options: tagOptions.map((o) => ({ value: o.value, label: o.label })),
      });
    }

    return next;
  }, [
    tr,
    symbolOptions,
    onToggleTradeStatus,
    marketOptions,
    onMarketsChange,
    tagOptions,
    onTagIdsChange,
  ]);

  const filters = useMemo(() => {
    const next: Filter<TradesFilterValue>[] = [];
    if (symbols.length > 0) {
      next.push(stableFilter("symbol", ops.symbol ?? "is_any_of", symbols));
    }
    if (tradeStatus) {
      next.push(stableFilter("status", ops.status ?? "is", [tradeStatus]));
    }
    if (markets && markets.length > 0) {
      next.push(stableFilter("market", ops.market ?? "is_any_of", markets));
    }
    if (tagIds && tagIds.length > 0) {
      next.push(stableFilter("tags", ops.tags ?? "is_any_of", tagIds));
    }
    return next;
  }, [symbols, tradeStatus, markets, tagIds, ops]);

  function handleChange(next: Filter<TradesFilterValue>[]) {
    setOps({
      symbol: operatorOf(next, "symbol", "is_any_of"),
      status: operatorOf(next, "status", "is"),
      market: operatorOf(next, "market", "is_any_of"),
      tags: operatorOf(next, "tags", "is_any_of"),
    });

    const nextSymbols = valuesOf(next, "symbol");
    onSymbolsChange(nextSymbols.length ? nextSymbols : undefined);

    if (onToggleTradeStatus) {
      const nextStatus = valuesOf(next, "status")[0] as TradeStatusFilter | undefined;
      if (!nextStatus) {
        if (tradeStatus) {
          if (onClearStatus) onClearStatus();
          else onToggleTradeStatus(tradeStatus);
        }
      } else if (nextStatus !== tradeStatus) {
        onToggleTradeStatus(nextStatus);
      }
    }

    if (onMarketsChange) {
      const nextMarkets = valuesOf(next, "market");
      onMarketsChange(nextMarkets.length ? nextMarkets : undefined);
    }

    if (onTagIdsChange) {
      const nextTags = valuesOf(next, "tags");
      onTagIdsChange(nextTags.length ? nextTags : undefined);
    }
  }

  return (
    <Filters
      filters={filters}
      fields={fields}
      onChange={handleChange}
      size="sm"
      allowMultiple={false}
      showSearchInput
      // The stock trigger renders at the Button default size (h-9 on phones);
      // supplying our own keeps it level with Created At / Sort.
      trigger={
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-label={tr({ id: "trades.addFilter", message: "Add filter" })}
          className={cn(triggerClassName, iconOnly && "w-8 px-0 [&>svg]:mx-0")}
        >
          <ListFilterPlus size={14} strokeWidth={1.75} />
          {iconOnly ? null : tr({ id: "trades.addFilter", message: "Add filter" })}
        </Button>
      }
      i18n={{
        addFilter: tr({ id: "trades.addFilter", message: "Add filter" }),
        searchFields: tr({ id: "trades.searchFilters", message: "Search filters…" }),
        noResultsFound: tr({ id: "trades.noSymbols", message: "No symbols found." }),
      }}
    />
  );
}
