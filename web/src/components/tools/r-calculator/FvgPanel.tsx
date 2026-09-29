import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { money, shares as fmtShares, signedMoney } from "@/lib/r-calculator/format";
import { useFvgStore } from "@/lib/r-calculator/useFvgStore";
import { cn } from "@/lib/cn";
import { Card } from "@/components/Card";
import { SegmentedControl } from "@/components/SegmentedControl";
import { CalcInputField } from "./CalcInputField";
import { TradeTicket, type TradeTicketRow } from "./TradeTicket";
import { WarningBanner } from "./WarningBanner";

export function FvgPanel() {
  const { t: localize } = useSecondaryLingui();

  const store = useFvgStore();
  const session = store.sessions.find((s) => s.id === store.activeId);
  if (!session) return null;

  const { result } = store;
  const isManual = session.entryAt === "manual";
  const long = session.direction === "long";

  const ticketRows: TradeTicketRow[] = [
    {
      label: localize({ id: "imports.direction", message: "Direction" }),
      value: long
        ? localize({ id: "trades.long", message: "Long" })
        : localize({ id: "trades.short", message: "Short" }),
    },
    {
      label: localize({ id: "market.entry", message: "Entry" }),
      value: `$${money(result.entryPrice)}`,
    },
    {
      label: localize({ id: "market.stop", message: "Stop" }),
      value: `$${money(result.stopPrice)}`,
    },
    {
      label: localize({ id: "market.target", message: "Target" }),
      value: `$${money(result.targetPrice)}`,
    },
    {
      label: localize({ id: "calculator.1rShare", message: "1R / share" }),
      value: `$${money(result.oneR)}`,
    },
    {
      label: localize({ id: "calculator.positionValue", message: "Position value" }),
      value: `$${money(result.positionValue)}`,
    },
    {
      label: localize({ id: "calculator.profitAtTarget", message: "Profit at target" }),
      value: `$${money(result.profitAtTarget)}`,
      tone: "profit",
    },
    {
      label: localize({ id: "calculator.lossAtStop", message: "Loss at stop" }),
      value: `$${money(result.lossAtStop)}`,
      tone: "loss",
    },
    {
      label: localize({ id: "calculator.realisedRR", message: "Realised R:R" }),
      value: `${result.realRR.toFixed(1)}:1`,
    },
  ];

  return (
    <div className="grid h-full min-h-0 grid-cols-1 gap-4 xl:grid-cols-2">
      <div className="flex min-h-0 flex-col gap-3 overflow-y-auto pr-0.5">
        <Card title={localize({ id: "calculator.tradeSetup", message: "Trade setup" })}>
          <div className="flex flex-col gap-3">
            <SegmentedControl
              ariaLabel={localize({ id: "calculator.tradeDirection", message: "Trade direction" })}
              fullWidth
              value={session.direction}
              onChange={(v) => store.setField("direction", v as "long" | "short")}
              options={[
                { value: "long", label: localize({ id: "calculator.long", message: "▲ Long" }) },
                { value: "short", label: localize({ id: "calculator.short", message: "▼ Short" }) },
              ]}
            />

            <div className="grid grid-cols-2 gap-2">
              <CalcInputField
                label={localize({ id: "calculator.gapTop", message: "Gap top" })}
                value={session.zoneTop}
                onValue={(n) => store.setField("zoneTop", n)}
                step={0.01}
                min={0}
                prefix="$"
              />
              <CalcInputField
                label={localize({ id: "calculator.gapBottom", message: "Gap bottom" })}
                value={session.zoneBottom}
                onValue={(n) => store.setField("zoneBottom", n)}
                step={0.01}
                min={0}
                prefix="$"
              />
            </div>

            <div className="flex flex-col gap-2">
              <span className="text-[13px] font-medium text-muted-foreground">
                {localize({ id: "calculator.entryAt", message: "Entry at" })}
              </span>
              <SegmentedControl
                ariaLabel={localize({ id: "calculator.entryLocation", message: "Entry location" })}
                fullWidth
                value={session.entryAt}
                onChange={(v) =>
                  store.setField("entryAt", v as "top" | "mid" | "bottom" | "manual")
                }
                options={[
                  { value: "top", label: localize({ id: "calculator.top", message: "Top" }) },
                  { value: "mid", label: localize({ id: "calculator.mid", message: "Mid" }) },
                  {
                    value: "bottom",
                    label: localize({ id: "calculator.bottom", message: "Bottom" }),
                  },
                  { value: "manual", label: localize({ id: "market.manual", message: "Manual" }) },
                ]}
              />
              {isManual ? (
                <CalcInputField
                  label={localize({ id: "calculator.entryPrice", message: "Entry price" })}
                  value={session.entryPrice}
                  onValue={(n) => store.setField("entryPrice", n)}
                  step={0.01}
                  min={0}
                  prefix="$"
                />
              ) : null}
            </div>

            <div className="grid grid-cols-2 gap-2">
              <CalcInputField
                label={localize({ id: "calculator.stopBuffer", message: "Stop buffer" })}
                value={session.stopBuffer}
                onValue={(n) => store.setField("stopBuffer", n)}
                step={0.01}
                min={0}
                prefix="$"
                hint={localize({ id: "calculator.beyondTheGap", message: "beyond the gap" })}
              />
              <CalcInputField
                label={localize({ id: "market.target", message: "Target" })}
                value={session.rMultiple}
                onValue={(n) => store.setField("rMultiple", n)}
                step={0.5}
                min={0}
                suffix="R"
                hint={localize({ id: "calculator.rewardMultiple", message: "reward multiple" })}
              />
              <CalcInputField
                label={localize({ id: "market.account", message: "Account" })}
                value={session.account}
                onValue={(n) => store.setField("account", n)}
                step={100}
                min={0}
                prefix="$"
              />
              <CalcInputField
                label={localize({ id: "trades.risk", message: "Risk" })}
                value={session.riskPct}
                onValue={(n) => store.setField("riskPct", n)}
                step={0.25}
                min={0}
                suffix="%"
                hint={localize({ id: "accounts.perTrade", message: "per trade" })}
              />
            </div>
          </div>
        </Card>

        <WarningBanner warns={store.warns} />
      </div>

      <div className="flex min-h-0 flex-col gap-3 xl:min-h-0">
        <TradeTicket
          heroLabel={localize({ id: "calculator.suggestedSize", message: "Suggested size" })}
          heroValue={fmtShares(result.shares)}
          heroUnit={localize({ id: "calculator.shareUnit", message: "sh" })}
          rows={ticketRows}
        />
        {/* Keep the card mounted when the inputs are invalid so the column doesn't collapse. */}
        <Card
          title={localize({ id: "calculator.riskRewardAxis", message: "Risk / reward axis" })}
          fill
          flush
          className="min-h-[280px]"
        >
          {result.valid ? (
            <FvgAxis
              long={long}
              rMultiple={session.rMultiple}
              entryPrice={result.entryPrice}
              stopPrice={result.stopPrice}
              targetPrice={result.targetPrice}
              profitAtTarget={result.profitAtTarget}
              lossAtStop={result.lossAtStop}
            />
          ) : (
            <div className="flex min-h-[240px] flex-1 items-center justify-center px-4 pb-4 text-center text-[13px] text-muted-foreground">
              {localize({
                id: "calculator.setAValidGapAndStopToPlotTheAxis",
                message: "Set a valid gap and stop to plot the axis.",
              })}
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}

function FvgAxis({
  long,
  rMultiple,
  entryPrice,
  stopPrice,
  targetPrice,
  profitAtTarget,
  lossAtStop,
}: {
  long: boolean;
  rMultiple: number;
  entryPrice: number;
  stopPrice: number;
  targetPrice: number;
  profitAtTarget: number;
  lossAtStop: number;
}) {
  const { t: localize } = useSecondaryLingui();

  const entryTop = (rMultiple / (rMultiple + 1)) * 100;

  return (
    <div className="flex min-h-0 flex-1 flex-col px-4 pb-4">
      <div className="relative flex min-h-[240px] flex-1 items-stretch justify-center gap-2 py-2">
        {/* Left scale */}
        <div className="relative w-10 shrink-0">
          <AxisLabel top="0%" text={`+${rMultiple}R`} tone="profit" align="right" />
          <AxisLabel top={`${entryTop}%`} text="0" tone="muted" align="right" emphasize />
          <AxisLabel top="100%" text="−1R" tone="loss" align="right" />
        </div>

        {/* Track */}
        <div className="relative w-12 shrink-0">
          <div className="absolute top-0 left-1/2 h-full w-px -translate-x-1/2 bg-border" />
          <div
            className="absolute top-0 left-1/2 w-[14px] origin-center -translate-x-1/2"
            style={{ height: "100%" }}
          >
            <div
              className="absolute inset-x-0 top-0 origin-bottom overflow-hidden rounded-t-full bg-linear-to-t from-profit/30 to-profit/90"
              style={{ height: `${entryTop}%` }}
            />
            <div
              className="absolute inset-x-0 origin-top overflow-hidden rounded-b-full bg-linear-to-b from-destructive/90 to-destructive/30"
              style={{
                top: `${entryTop}%`,
                height: `${100 - entryTop}%`,
              }}
            />
          </div>
          <div
            aria-hidden
            className={cn(
              "absolute left-1/2 z-10 flex size-5 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-background text-[9px] font-bold ring-2",
              long ? "text-profit ring-profit" : "text-loss ring-loss",
            )}
            style={{ top: `${entryTop}%` }}
          >
            {long ? "▲" : "▼"}
          </div>
        </div>

        {/* Right prices */}
        <div className="relative w-28 shrink-0 sm:w-32">
          <FvgPriceLabel
            top="0%"
            tone="profit"
            caption={localize({
              id: "calculator.targetValue0R",
              message: `Target +${{ value0: rMultiple }}R`,
            })}
            price={money(targetPrice)}
            pl={signedMoney(profitAtTarget)}
          />
          <FvgPriceLabel
            top={`${entryTop}%`}
            tone="text"
            caption={localize({ id: "market.entry", message: "Entry" })}
            price={money(entryPrice)}
            emphasize
          />
          <FvgPriceLabel
            top="100%"
            tone="loss"
            caption={localize({ id: "calculator.stopMinus1R", message: "Stop −1R" })}
            price={money(stopPrice)}
            pl={signedMoney(-lossAtStop)}
          />
        </div>
      </div>
    </div>
  );
}

function FvgPriceLabel({
  top,
  tone,
  caption,
  price,
  pl,
  emphasize,
}: {
  top: string;
  tone: "profit" | "loss" | "text";
  caption: string;
  price: string;
  pl?: string;
  emphasize?: boolean;
}) {
  return (
    <div className="absolute left-0 -translate-y-1/2" style={{ top }}>
      <p className="m-0 text-[11px] text-muted-foreground">{caption}</p>
      <p
        className={cn(
          "m-0 tabular-nums",
          emphasize ? "text-sm font-semibold" : "text-[13px] font-medium",
          tone === "profit" && "text-profit",
          tone === "loss" && "text-loss",
          tone === "text" && "text-foreground",
        )}
      >
        {price}
      </p>
      {pl ? (
        <p
          className={cn(
            "m-0 text-[11px] tabular-nums",
            tone === "profit" ? "text-profit/80" : "text-loss/80",
          )}
        >
          {pl}
        </p>
      ) : null}
    </div>
  );
}

function AxisLabel({
  top,
  text,
  tone,
  align,
  emphasize,
}: {
  top: string;
  text: string;
  tone: "profit" | "loss" | "muted";
  align: "left" | "right";
  emphasize?: boolean;
}) {
  return (
    <span
      className={cn(
        "absolute -translate-y-1/2 whitespace-nowrap text-[10px] tabular-nums",
        align === "right" ? "right-0 text-right" : "left-0 text-left",
        tone === "profit" && "text-profit",
        tone === "loss" && "text-loss",
        tone === "muted" && "text-muted-foreground",
        emphasize && "font-semibold",
      )}
      style={{ top }}
    >
      {text}
    </span>
  );
}
