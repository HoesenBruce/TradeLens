import { t as localize } from "@lingui/core/macro";
import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import type { AvgMode, PnlMode, UnitMode } from "./ReportsDisplayContext";
import { SegmentedControl } from "./SegmentedControl";

export type ReportsSide = "all" | "long" | "short";
export type ReportsDuration = "all" | "scalp" | "day" | "swing";

export interface ReportsControlBarProps {
  side: ReportsSide;
  duration: ReportsDuration;
  onSideChange: (s: ReportsSide) => void;
  onDurationChange: (d: ReportsDuration) => void;
  pnlMode: PnlMode;
  unitMode: UnitMode;
  avgMode: AvgMode;
  onPnlModeChange: (m: PnlMode) => void;
  onUnitModeChange: (m: UnitMode) => void;
  onAvgModeChange: (m: AvgMode) => void;
  pctEnabled: boolean;
}

const SIDE_OPTS = [
  {
    value: "all",
    get label() {
      return localize({ id: "reports.all", message: "All" });
    },
  },
  {
    value: "long",
    get label() {
      return localize({ id: "trades.long", message: "Long" });
    },
  },
  {
    value: "short",
    get label() {
      return localize({ id: "trades.short", message: "Short" });
    },
  },
];

const DURATION_OPTS = [
  {
    value: "all",
    get label() {
      return localize({ id: "reports.all", message: "All" });
    },
  },
  {
    value: "scalp",
    get label() {
      return localize({ id: "reports.scalp", message: "Scalp" });
    },
  },
  {
    value: "day",
    get label() {
      return localize({ id: "reports.intraday", message: "Day" });
    },
  },
  {
    value: "swing",
    get label() {
      return localize({ id: "reports.swing", message: "Swing" });
    },
  },
];

const PNL_OPTS = [
  {
    value: "net",
    get label() {
      return localize({ id: "accounts.net", message: "Net" });
    },
  },
  {
    value: "gross",
    get label() {
      return localize({ id: "accounts.gross", message: "Gross" });
    },
  },
];

const UNIT_OPTS = [
  { value: "abs", label: "$" },
  { value: "pct", label: "%" },
];

const AVG_OPTS = [
  {
    value: "mean",
    get label() {
      return localize({ id: "reports.mean", message: "Mean" });
    },
  },
  {
    value: "median",
    get label() {
      return localize({ id: "reports.median", message: "Median" });
    },
  },
];

export function ReportsControlBar({
  side,
  duration,
  onSideChange,
  onDurationChange,
  pnlMode,
  unitMode,
  avgMode,
  onPnlModeChange,
  onUnitModeChange,
  onAvgModeChange,
  pctEnabled,
}: ReportsControlBarProps) {
  const { t: localize } = useSecondaryLingui();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <SegmentedControl
        ariaLabel={localize({ id: "imports.fieldSide", message: "Side" })}
        size="xs"
        options={SIDE_OPTS}
        value={side}
        onChange={(v) => onSideChange(v as ReportsSide)}
      />
      <SegmentedControl
        ariaLabel={localize({ id: "reports.duration", message: "Duration" })}
        size="xs"
        options={DURATION_OPTS}
        value={duration}
        onChange={(v) => onDurationChange(v as ReportsDuration)}
      />
      <SegmentedControl
        ariaLabel={localize({ id: "reports.pLBasis", message: "P&L basis" })}
        size="xs"
        options={PNL_OPTS}
        value={pnlMode}
        onChange={(v) => onPnlModeChange(v as PnlMode)}
      />
      <div
        title={
          pctEnabled
            ? undefined
            : localize({
                id: "reports.aFundedScopeIsRequiredToView",
                message: "A funded scope is required to view %",
              })
        }
      >
        <SegmentedControl
          ariaLabel={localize({ id: "reports.unit", message: "Unit" })}
          size="xs"
          options={pctEnabled ? UNIT_OPTS : UNIT_OPTS.filter((o) => o.value === "abs")}
          value={pctEnabled ? unitMode : "abs"}
          onChange={(v) => onUnitModeChange(v as UnitMode)}
        />
      </div>
      <SegmentedControl
        ariaLabel={localize({ id: "reports.averageBasis", message: "Average basis" })}
        size="xs"
        options={AVG_OPTS}
        value={avgMode}
        onChange={(v) => onAvgModeChange(v as AvgMode)}
      />
    </div>
  );
}
