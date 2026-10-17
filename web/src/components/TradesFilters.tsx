import { useLingui } from "@lingui/react/macro";
import {
  ArrowUpDown,
  ChartCandlestick,
  CircleDot,
  Clock,
  ListFilterPlus,
  Search,
  Tags,
  Wallet,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import { Filters, type FilterFieldConfig, type FilterOperator } from "@/components/filters";
import type {
  MarketFacetOption,
  SymbolFacetOption,
  TagFacetOption,
  TradeFilter,
} from "@/lib/tradeFilters";

export interface TradesFiltersProps {
  triggerClassName?: string;
  iconOnly?: boolean;
  conditions: TradeFilter[];
  onConditionsChange: (filters: TradeFilter[]) => void;
  symbolOptions?: SymbolFacetOption[];
  marketOptions?: MarketFacetOption[];
  tagOptions?: TagFacetOption[];
}

export function TradesFilters({
  triggerClassName,
  iconOnly = false,
  conditions,
  onConditionsChange,
  symbolOptions = [],
  marketOptions = [],
  tagOptions = [],
}: TradesFiltersProps) {
  const { t: tr } = useLingui();
  const operators: FilterOperator[] = [
    { value: "is_any_of", label: tr({ id: "trades.filterInclude", message: "Include" }) },
    { value: "is_not_any_of", label: tr({ id: "trades.filterExclude", message: "Exclude" }) },
  ];
  const fields: FilterFieldConfig<string>[] = [
    {
      key: "symbol",
      label: tr({ id: "trades.symbol", message: "Symbol" }),
      icon: <Search />,
      searchable: true,
      options: symbolOptions,
    },
    {
      key: "result",
      label: tr({ id: "trades.tradeResult", message: "Trade result" }),
      icon: <CircleDot />,
      options: [
        { value: "win", label: tr({ id: "trades.win", message: "Win" }) },
        { value: "loss", label: tr({ id: "trades.loss", message: "Loss" }) },
        { value: "wash", label: tr({ id: "trades.breakEven", message: "Break-even" }) },
        { value: "unknown", label: tr({ id: "trades.resultUnknown", message: "Unknown result" }) },
      ],
    },
    {
      key: "direction",
      label: tr({ id: "trades.direction", message: "Direction" }),
      icon: <ArrowUpDown />,
      options: [
        { value: "long", label: tr({ id: "trades.long", message: "Long" }) },
        { value: "short", label: tr({ id: "trades.short", message: "Short" }) },
        {
          value: "unknown",
          label: tr({ id: "trades.directionUnknown", message: "Unknown direction" }),
        },
      ],
    },
    {
      key: "progress",
      label: tr({ id: "trades.closeProgress", message: "Close progress" }),
      icon: <CircleDot />,
      options: [
        { value: "open", label: tr({ id: "trades.lifecycleOpen", message: "Position open" }) },
        {
          value: "partially_closed",
          label: tr({ id: "trades.lifecyclePartial", message: "Partially closed" }),
        },
        {
          value: "closed",
          label: tr({ id: "trades.lifecycleClosed", message: "Position closed" }),
        },
        {
          value: "unknown",
          label: tr({ id: "trades.lifecycleUnknown", message: "Position unknown" }),
        },
      ],
    },
    {
      key: "position_type",
      label: tr({ id: "trades.positionTypeFilter", message: "Position type" }),
      icon: <Wallet />,
      options: [
        { value: "cash", label: tr({ id: "trades.positionCash", message: "Cash" }) },
        { value: "margin", label: tr({ id: "trades.positionMargin", message: "Margin" }) },
        {
          value: "unknown",
          label: tr({ id: "trades.positionTypeUnknown", message: "Type unknown" }),
        },
      ],
    },
    {
      key: "duration",
      label: tr({ id: "trades.holdingDurationFilter", message: "Holding duration" }),
      icon: <Clock />,
      options: [
        {
          value: "elapsed_under_5m",
          label: tr({ id: "trades.durationUnder5", message: "Elapsed: under 5 minutes" }),
        },
        {
          value: "elapsed_5_30m",
          label: tr({ id: "trades.duration5To30", message: "Elapsed: 5–30 minutes" }),
        },
        {
          value: "elapsed_30_60m",
          label: tr({ id: "trades.duration30To60", message: "Elapsed: 30–60 minutes" }),
        },
        {
          value: "elapsed_1_4h",
          label: tr({ id: "trades.duration1To4", message: "Elapsed: 1–4 hours" }),
        },
        {
          value: "elapsed_4h_plus",
          label: tr({ id: "trades.duration4Plus", message: "Elapsed: 4 hours or more" }),
        },
        {
          value: "calendar_0",
          label: tr({ id: "trades.durationSameDay", message: "Calendar: same day" }),
        },
        {
          value: "calendar_1",
          label: tr({ id: "trades.duration1Day", message: "Calendar: 1 day" }),
        },
        {
          value: "calendar_2_5",
          label: tr({ id: "trades.duration2To5Days", message: "Calendar: 2–5 days" }),
        },
        {
          value: "calendar_6_plus",
          label: tr({ id: "trades.duration6Plus", message: "Calendar: 6 days or more" }),
        },
        {
          value: "unknown",
          label: tr({ id: "trades.holdingUnknown", message: "Holding duration is unknown." }),
        },
      ],
    },
    {
      key: "market",
      label: tr({ id: "trades.market", message: "Market" }),
      icon: <ChartCandlestick />,
      searchable: true,
      options: marketOptions,
    },
    {
      key: "tags",
      label: tr({ id: "trades.tags", message: "Tags" }),
      icon: <Tags />,
      searchable: true,
      options: tagOptions,
      operators: [
        ...operators,
        { value: "includes_all", label: tr({ id: "trades.includesAll", message: "includes all" }) },
      ],
    },
  ].map((field) => {
    const title = field.label;
    return {
      type: "multiselect",
      searchable: false,
      operators,
      ...field,
      removeLabel: tr({ id: "common.clearNamedFilter", message: `Clear ${title} filter` }),
    };
  });

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Filters
        filters={conditions}
        fields={fields}
        onChange={(next) => onConditionsChange(next.filter((f) => f.values.length > 0))}
        size="sm"
        allowMultiple={false}
        showSearchInput
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
          selectedCount: tr({ id: "trades.filterSelected", message: "selected" }),
          addFilter: tr({ id: "trades.addFilter", message: "Add filter" }),
          searchFields: tr({ id: "trades.searchFilters", message: "Search filters…" }),
          noResultsFound: tr({ id: "trades.filterNoOptions", message: "No matching options." }),
        }}
      />
      {conditions.some((f) => f.field === "duration") && (
        <p className="w-full text-xs text-muted-foreground">
          {tr({
            id: "trades.durationFilterHint",
            message:
              "Elapsed ranges use precise timestamps; calendar ranges use date-only records in the market time zone. Open positions run to now.",
          })}
        </p>
      )}
      {conditions.some((f) => f.field === "result") && (
        <p className="w-full text-xs text-muted-foreground">
          {tr({
            id: "trades.resultFilterHint",
            message:
              "Trade results include completed trades only, including when excluding a result.",
          })}
        </p>
      )}
    </div>
  );
}
