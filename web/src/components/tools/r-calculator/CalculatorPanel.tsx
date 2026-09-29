import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { money, shares as fmtShares } from "@/lib/r-calculator/format";
import { limiterLabel } from "@/lib/r-calculator/messages";
import { useRCalculatorStore } from "@/lib/r-calculator/useRCalculatorStore";
import { Card } from "@/components/Card";
import { SegmentedControl } from "@/components/SegmentedControl";
import { CalcInputField } from "./CalcInputField";
import { ExitLadder } from "./ExitLadder";
import { RAxis } from "./RAxis";
import { TradeTicket, type TradeTicketRow } from "./TradeTicket";
import { WarningBanner } from "./WarningBanner";

export function CalculatorPanel() {
  const { t: localize } = useSecondaryLingui();

  const store = useRCalculatorStore();
  const session = store.sessions.find((s) => s.id === store.activeId);
  if (!session) return null;

  const { input, result } = store;
  const isOpt = session.instrument === "options";
  const unit = isOpt
    ? localize({ id: "calculator.contractUnit", message: "ct" })
    : localize({ id: "calculator.shareUnit", message: "sh" });
  const long = input.direction === "long";

  const ticketRows: TradeTicketRow[] = [
    {
      label: isOpt
        ? localize({ id: "accounts.type", message: "Type" })
        : localize({ id: "imports.direction", message: "Direction" }),
      value: isOpt
        ? session.optionType === "call"
          ? localize({ id: "calculator.call2", message: "Call" })
          : localize({ id: "calculator.put2", message: "Put" })
        : long
          ? localize({ id: "trades.long", message: "Long" })
          : localize({ id: "trades.short", message: "Short" }),
    },
    { label: localize({ id: "market.entry", message: "Entry" }), value: `$${money(input.entry)}` },
    {
      label: localize({ id: "market.stop", message: "Stop" }),
      value: `$${money(result.stopPrice)}`,
    },
    {
      label: isOpt
        ? localize({ id: "calculator.premiumCost", message: "Premium cost" })
        : localize({ id: "calculator.positionValue", message: "Position value" }),
      value: `$${money(result.posValue)}`,
    },
    {
      label: localize({ id: "calculator.riskBudget", message: "Risk budget" }),
      value: `$${money(result.riskAmt)}`,
    },
    {
      label: localize({ id: "calculator.cashLeft", message: "Cash left" }),
      value: `$${money(result.remainingCash)}`,
    },
  ];

  if (isOpt) {
    ticketRows.push({
      label: localize({ id: "calculator.maxLoss", message: "Max loss" }),
      value: `$${money(result.maxLoss)}`,
      tone: "loss",
    });
  }

  if (result.limiter !== "none") {
    ticketRows.push({
      label: localize({ id: "calculator.limiter", message: "Limiter" }),
      value: limiterLabel(result.limiter),
    });
  }

  return (
    <div className="grid h-full min-h-0 grid-cols-1 gap-4 xl:grid-cols-2">
      <div className="flex min-h-0 flex-col gap-3 overflow-y-auto pr-0.5">
        <Card title={localize({ id: "calculator.tradeSetup", message: "Trade setup" })}>
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-2 gap-2">
              <SegmentedControl
                ariaLabel={localize({
                  id: "imports.fieldInstrumentType",
                  message: "Instrument type",
                })}
                fullWidth
                value={session.instrument}
                onChange={(v) => store.setField("instrument", v as "stock" | "options")}
                options={[
                  {
                    value: "stock",
                    label: localize({ id: "calculator.stockShares", message: "Stock shares" }),
                  },
                  {
                    value: "options",
                    label: localize({
                      id: "calculator.optionsContracts",
                      message: "Options contracts",
                    }),
                  },
                ]}
              />
              {isOpt ? (
                <SegmentedControl
                  ariaLabel={localize({ id: "calculator.optionType", message: "Option type" })}
                  fullWidth
                  value={session.optionType}
                  onChange={(v) => store.setField("optionType", v as "call" | "put")}
                  options={[
                    {
                      value: "call",
                      label: localize({ id: "calculator.call", message: "▲ Call" }),
                    },
                    { value: "put", label: localize({ id: "calculator.put", message: "▼ Put" }) },
                  ]}
                />
              ) : (
                <SegmentedControl
                  ariaLabel={localize({
                    id: "calculator.tradeDirection",
                    message: "Trade direction",
                  })}
                  fullWidth
                  value={session.direction}
                  onChange={(v) => store.setField("direction", v as "long" | "short")}
                  options={[
                    {
                      value: "long",
                      label: localize({ id: "calculator.long", message: "▲ Long" }),
                    },
                    {
                      value: "short",
                      label: localize({ id: "calculator.short", message: "▼ Short" }),
                    },
                  ]}
                />
              )}
            </div>

            <div className="grid grid-cols-2 gap-2 sm:grid-cols-2">
              {isOpt ? (
                <>
                  <CalcInputField
                    label={localize({ id: "calculator.entryPremium", message: "Entry premium" })}
                    value={session.entryPrem}
                    onValue={(n) => store.setField("entryPrem", n)}
                    step={0.05}
                    min={0}
                    prefix="$"
                  />
                  <CalcInputField
                    label={localize({ id: "calculator.stopPremium", message: "Stop premium" })}
                    value={session.stopPrem}
                    onValue={(n) => store.setField("stopPrem", n)}
                    step={0.05}
                    min={0}
                    prefix="$"
                  />
                </>
              ) : (
                <>
                  <CalcInputField
                    label={localize({ id: "market.entry", message: "Entry" })}
                    value={session.entry}
                    onValue={(n) => store.setField("entry", n)}
                    step={0.01}
                    min={0}
                    prefix="$"
                  />
                  <CalcInputField
                    label={localize({ id: "market.stop", message: "Stop" })}
                    value={session.stop}
                    onValue={(n) => store.setField("stop", n)}
                    step={0.01}
                    min={0}
                    prefix="$"
                  />
                </>
              )}
              <CalcInputField
                label={localize({ id: "calculator.capital", message: "Capital" })}
                value={session.capital}
                onValue={(n) => store.setField("capital", n)}
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
              {isOpt ? (
                <div className="col-span-2">
                  <CalcInputField
                    label={localize({
                      id: "calculator.sharesContract",
                      message: "Shares / contract",
                    })}
                    value={session.contractSize}
                    onValue={(n) => store.setField("contractSize", n)}
                    step={1}
                    min={1}
                    suffix={localize({ id: "calculator.shareUnit", message: "sh" })}
                    hint={localize({ id: "trades.multiplier", message: "Multiplier" })}
                  />
                </div>
              ) : null}
            </div>
          </div>
        </Card>

        <ExitLadder />
        <WarningBanner warns={[...store.warns, ...store.exitWarns]} />
      </div>

      <div className="flex min-h-0 flex-col gap-3 xl:min-h-0">
        <TradeTicket
          heroLabel={localize({ id: "calculator.suggestedSize", message: "Suggested size" })}
          heroValue={fmtShares(result.shares)}
          heroUnit={unit}
          rows={ticketRows}
        />
        <Card
          title={localize({ id: "calculator.riskRewardAxis", message: "Risk / reward axis" })}
          fill
          flush
          className="min-h-[280px]"
        >
          <RAxis />
        </Card>
      </div>
    </div>
  );
}
