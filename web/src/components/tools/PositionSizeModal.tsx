import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { Link } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { soleAccountId, useFilters } from "@/lib/filters";
import { useAccounts } from "@/lib/hooks/useAccounts";
import { useRiskRules } from "@/lib/hooks/useRiskRules";
import { positionSizeFromRisk } from "@/lib/positionSize";
import { Modal } from "@/components/Modal";
import { fieldInputClass } from "@/components/field-styles";

const labelClass =
  "mb-1 block text-[10px] font-medium uppercase tracking-widest text-muted-foreground";
const inputClass = fieldInputClass;

export function PositionSizeModal({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t: localize } = useSecondaryLingui();

  const accountId = useFilters((s) => soleAccountId(s.accountIds));
  const accounts = useAccounts().data ?? [];
  const account = accounts.find((a) => a.id === accountId) ?? accounts[0];
  const riskRules = useRiskRules().data;
  const defaultPct = riskRules?.default_account_risk_pct ?? 1;

  const [equity, setEquity] = useState("10000");
  const [riskPct, setRiskPct] = useState("1");
  const [entry, setEntry] = useState("");
  const [stop, setStop] = useState("");

  useEffect(() => {
    if (account) setEquity(String(account.starting_balance));
  }, [account]);

  useEffect(() => {
    if (defaultPct != null) setRiskPct(String(defaultPct));
  }, [defaultPct]);

  const result = useMemo(() => {
    const eq = Number(equity);
    const pct = Number(riskPct);
    const en = Number(entry);
    const st = Number(stop);
    if (![eq, pct, en, st].every((n) => Number.isFinite(n))) return null;
    return positionSizeFromRisk({
      equity: eq,
      riskPct: pct,
      entryPrice: en,
      stopPrice: st,
    });
  }, [equity, riskPct, entry, stop]);

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={localize({ id: "market.size", message: "Position size" })}
      className="max-w-[420px]"
    >
      <p className="m-0 text-xs leading-relaxed text-muted-foreground">
        {localize({
          id: "calculator.sizeFromAccountEquityRiskEntryAndStopUsesYourDefaultRisk",
          message:
            "Size from account equity, risk %, entry, and stop. Uses your default risk % from Settings when available.",
        })}
      </p>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass} htmlFor="ps-equity">
            {localize({ id: "calculator.equity", message: "Equity ($)" })}
          </label>
          <input
            id="ps-equity"
            className={inputClass}
            inputMode="decimal"
            value={equity}
            onChange={(e) => setEquity(e.target.value)}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="ps-pct">
            {localize({ id: "calculator.risk", message: "Risk %" })}
          </label>
          <input
            id="ps-pct"
            className={inputClass}
            inputMode="decimal"
            value={riskPct}
            onChange={(e) => setRiskPct(e.target.value)}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="ps-entry">
            {localize({ id: "market.entry", message: "Entry" })}
          </label>
          <input
            id="ps-entry"
            className={inputClass}
            inputMode="decimal"
            value={entry}
            onChange={(e) => setEntry(e.target.value)}
            placeholder="100"
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="ps-stop">
            {localize({ id: "market.stop", message: "Stop" })}
          </label>
          <input
            id="ps-stop"
            className={inputClass}
            inputMode="decimal"
            value={stop}
            onChange={(e) => setStop(e.target.value)}
            placeholder="99"
          />
        </div>
      </div>
      {result ? (
        <div className="rounded-panel border border-border bg-muted px-3.5 py-3">
          <p className="m-0 text-[10px] font-medium uppercase tracking-widest text-muted-foreground">
            {localize({ id: "calculator.suggestedSize", message: "Suggested size" })}
          </p>
          <p className="mt-1 mb-0 text-2xl tabular-nums text-foreground">
            {result.qty} {localize({ id: "calculator.shares", message: "shares" })}
          </p>
          <p className="mt-1 mb-0 text-[11px] text-muted-foreground">
            {localize({ id: "calculator.risk2", message: "Risk $" })}
            {result.riskDollars.toFixed(2)} · ${result.perShareRisk.toFixed(2)}{" "}
            {localize({ id: "calculator.share", message: "/ share" })}
          </p>
        </div>
      ) : (
        <p className="m-0 text-xs text-muted-foreground">
          {localize({
            id: "calculator.enterEquityRiskEntryAndStopToSeeSize",
            message: "Enter equity, risk %, entry, and stop to see size.",
          })}
        </p>
      )}
      <Link
        to="/calculator"
        onClick={() => onOpenChange(false)}
        className="mt-1 inline-flex text-[11px] font-medium text-primary no-underline transition-colors hover:text-foreground"
      >
        {localize({
          id: "calculator.openFullPlannerExitLadderRAxis",
          message: "Open full planner — exit ladder & R-axis →",
        })}
      </Link>
    </Modal>
  );
}
