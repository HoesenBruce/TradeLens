import { Trans } from "@lingui/react/macro";
import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { Link } from "@tanstack/react-router";
import type { ComplianceReport } from "@/lib/api/types";
import { usePrivacyMode } from "@/lib/displayPrefs";
import { fmtDayShort } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { Card } from "./Card";
import { EmptyState } from "./EmptyState";
import { Pill } from "./Pill";
import { useReportsMoney } from "./ReportsDisplayContext";
import { Skeleton } from "./Skeleton";
import { StatCard } from "./StatCard";

export interface ReportsRuleComplianceProps {
  report?: ComplianceReport;
  loading: boolean;
  error: boolean;
}

/**
 * "What do the rules cost me when I break them?" — compares P&L on days that
 * followed the risk rules against days that broke them, the single most
 * persuasive discipline chart a journal can show.
 */
export function ReportsRuleCompliance({ report, loading, error }: ReportsRuleComplianceProps) {
  const { t: localize } = useSecondaryLingui();

  usePrivacyMode();
  const money = useReportsMoney();
  const locale = intlLocale();

  if (loading) {
    return (
      <Card title={localize({ id: "reports.ruleCompliance", message: "Rule compliance" })}>
        <Skeleton height="160px" />
      </Card>
    );
  }
  if (error) {
    return (
      <Card title={localize({ id: "reports.ruleCompliance", message: "Rule compliance" })}>
        <p className="m-0 text-xs text-destructive">
          {localize({
            id: "reports.failedToLoadCompliance",
            message: "Failed to load compliance.",
          })}
        </p>
      </Card>
    );
  }
  if (!report?.rules_configured) {
    return (
      <Card title={localize({ id: "reports.ruleCompliance", message: "Rule compliance" })}>
        <EmptyState
          title={localize({
            id: "reports.noRiskRulesConfigured",
            message: "No risk rules configured",
          })}
          hint={localize({
            id: "reports.setMaxRiskPerTradeAndMaxDailyLossInSettingsRules",
            message:
              "Set max risk per trade and max daily loss in Settings → Rules to score every trading day against your own process.",
          })}
        />
        <Link
          to="/settings"
          className="mt-2 self-start text-xs font-medium text-primary hover:underline"
        >
          {localize({ id: "reports.openSettings", message: "Open settings" })}
        </Link>
      </Card>
    );
  }

  const breachDays = report.days.filter((d) => !d.compliant);
  const recentBreaches = breachDays.slice(-6).reverse();
  const scoredDays = report.compliant_days + report.breach_days;
  const adherence = scoredDays > 0 ? (report.compliant_days / scoredDays) * 100 : null;

  return (
    <Card
      title={localize({ id: "reports.ruleCompliance", message: "Rule compliance" })}
      description={localize({
        id: "reports.daysScoredAgainstYourRiskRulesFollowingThemShouldShowUpIn",
        message:
          "Days scored against your risk rules — following them should show up in the P&L split.",
      })}
    >
      <div className="flex flex-col gap-5">
        <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
          <StatCard
            label={localize({ id: "reports.adherence", message: "Adherence" })}
            value={adherence != null ? `${adherence.toFixed(0)}%` : "—"}
            hint={localize({
              id: "reports.value0OfValue1Days",
              message: `${{ value0: report.compliant_days }} of ${{ value1: scoredDays }} days`,
            })}
          />
          <StatCard
            label={localize({ id: "reports.pLRulesFollowed", message: "P&L, rules followed" })}
            value={money.format(report.compliant_pnl)}
            accent={report.compliant_pnl >= 0 ? "pos" : "neg"}
          />
          <StatCard
            label={localize({ id: "reports.pLRulesBroken", message: "P&L, rules broken" })}
            value={money.format(report.breach_pnl)}
            accent={report.breach_pnl >= 0 ? "pos" : "neg"}
          />
          <StatCard
            label={localize({ id: "reports.violations", message: "Violations" })}
            value={String(
              report.risk_violations +
                report.daily_loss_breaches +
                report.trade_limit_breaches +
                report.loss_streak_breaches,
            )}
            hint={
              [
                report.risk_violations > 0 &&
                  localize({
                    id: "reports.value0OverRisked",
                    message: `${{ value0: report.risk_violations }} over-risked`,
                  }),
                report.daily_loss_breaches > 0 &&
                  localize({
                    id: "reports.value0DailyLoss",
                    message: `${{ value0: report.daily_loss_breaches }} daily-loss`,
                  }),
                report.trade_limit_breaches > 0 &&
                  localize({
                    id: "reports.value0OverTraded",
                    message: `${{ value0: report.trade_limit_breaches }} over-traded`,
                  }),
                report.loss_streak_breaches > 0 &&
                  localize({
                    id: "reports.value0LossStreak",
                    message: `${{ value0: report.loss_streak_breaches }} loss-streak`,
                  }),
              ]
                .filter(Boolean)
                .join(" \u00b7 ") || "none"
            }
          />
        </div>

        {report.unknown_risk > 0 && (
          <p className="m-0 text-[11px] leading-relaxed text-muted-foreground">
            <Trans id="reports.tradeHadNoRecordedInitialRiskAndCouldSentence">
              {report.unknown_risk} trade{report.unknown_risk === 1 ? "" : "s"} had no recorded
              initial risk and could not be scored against the per-trade risk rule.
            </Trans>
          </p>
        )}

        {recentBreaches.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <p className="m-0 text-[10px] font-semibold tracking-widest text-muted-foreground uppercase">
              {localize({ id: "reports.recentBreachDays", message: "Recent breach days" })}
            </p>
            <ul className="m-0 flex list-none flex-col gap-1 p-0">
              {recentBreaches.map((d) => (
                <li
                  key={d.date}
                  className="flex items-center justify-between gap-3 rounded-md px-2 py-1.5 text-[13px] hover:bg-accent"
                >
                  <span className="flex items-center gap-2">
                    <span className="tabular-nums text-foreground">
                      {fmtDayShort(`${d.date}T12:00:00Z`, locale)}
                    </span>
                    {d.risk_violations > 0 && (
                      <Pill tone="neg">
                        {localize({ id: "reports.overRisked", message: "over-risked ×" })}
                        {d.risk_violations}
                      </Pill>
                    )}
                    {d.daily_loss_breach && (
                      <Pill tone="neg">
                        {localize({ id: "reports.dailyLoss", message: "daily loss" })}
                      </Pill>
                    )}
                    {d.trade_limit_breach && (
                      <Pill tone="neg">
                        {localize({ id: "reports.overTraded", message: "over-traded" })}
                      </Pill>
                    )}
                    {d.loss_streak_breach && (
                      <Pill tone="neg">
                        {localize({ id: "reports.lossStreak", message: "loss streak" })}
                      </Pill>
                    )}
                  </span>
                  <span
                    className={
                      d.net_pnl >= 0
                        ? "tabular-nums font-medium text-success"
                        : "tabular-nums font-medium text-destructive"
                    }
                  >
                    {money.format(d.net_pnl)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </Card>
  );
}
