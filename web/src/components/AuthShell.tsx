import { useLingui } from "@lingui/react";
import { msg } from "@lingui/core/macro";
import type { MessageDescriptor } from "@lingui/core";
import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { type ReactNode, useId } from "react";
import { AppLogo } from "@/components/AppLogo";
import { BRAND } from "@/lib/brand";
import { cn } from "@/lib/cn";

/**
 * Equity curve at panel scale: a smooth run with one drawdown, drawn on load,
 * grounded by a gradient area fill. Journal memos pin onto its turning points
 * and a lens settles over the final review point. Motion runs once on entry.
 */
const EQUITY_LINE =
  "M0 270 C 50 266, 90 252, 130 236 C 160 224, 180 250, 214 248 C 250 246, 270 210, 305 182 C 330 162, 350 186, 380 178 C 420 168, 440 120, 480 96 C 505 81, 522 66, 540 60";

const EQUITY_AREA = `${EQUITY_LINE} L540 320 L0 320 Z`;

type Memo = {
  meta: string;
  note: MessageDescriptor;
  r: string;
  tone: "profit" | "loss";
  /** Chip anchor: % offsets + translate that hangs it off the marker. */
  chip: string;
  marker: { cx: number; cy: number };
  delay: number;
};

const MEMOS: Memo[] = [
  {
    meta: "MAR 14 · NVDA",
    note: msg({
      id: "auth.chasedTheOpenCutItFast",
      message: "Chased the open. Cut it fast.",
    }),
    r: "−1.4R",
    tone: "loss",
    chip: "left-[36.6%] top-[77.5%] -translate-x-1/2 translate-y-3",
    marker: { cx: 214, cy: 248 },
    delay: 900,
  },
  {
    meta: "APR 02 · ES",
    note: msg({ id: "auth.waitedForTheRetest", message: "Waited for the retest." }),
    r: "+2.1R",
    tone: "profit",
    chip: "left-[65.1%] top-[55.6%] -translate-x-1/2 translate-y-[calc(-100%_-_12px)]",
    marker: { cx: 380, cy: 178 },
    delay: 1800,
  },
  {
    meta: "MAY 21 · AAPL",
    note: msg({
      id: "auth.aSetupFollowedThePlan",
      message: "A+ setup. Followed the plan.",
    }),
    r: "+3.2R",
    tone: "profit",
    chip: "left-[82.2%] top-[30%] -translate-x-full translate-y-[calc(-100%_-_14px)]",
    marker: { cx: 480, cy: 96 },
    delay: 2700,
  },
];

function EquityArtwork() {
  const { _ } = useLingui();
  const uid = useId().replace(/:/g, "");
  const glowId = `${uid}-glow`;
  const areaId = `${uid}-area`;
  const fadeId = `${uid}-fade`;
  const maskId = `${uid}-mask`;

  return (
    <div aria-hidden className="pointer-events-none relative w-full max-w-[36rem] select-none">
      <svg viewBox="0 0 584 320" fill="none" className="w-full">
        <defs>
          <radialGradient id={glowId}>
            <stop offset="0" stopColor="var(--primary)" stopOpacity="0.28" />
            <stop offset="1" stopColor="var(--primary)" stopOpacity="0" />
          </radialGradient>
          <linearGradient id={areaId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="var(--primary)" stopOpacity="0.16" />
            <stop offset="0.85" stopColor="var(--primary)" stopOpacity="0" />
          </linearGradient>
          <linearGradient id={fadeId} x1="0" y1="0" x2="1" y2="0">
            <stop offset="0.7" stopColor="#fff" />
            <stop offset="1" stopColor="#fff" stopOpacity="0" />
          </linearGradient>
          <mask id={maskId}>
            <rect width="584" height="320" fill={`url(#${fadeId})`} />
          </mask>
        </defs>
        <path
          d={EQUITY_AREA}
          fill={`url(#${areaId})`}
          mask={`url(#${maskId})`}
          className="motion-safe:animate-[auth-panel-in_700ms_ease-out_both] motion-safe:[animation-delay:700ms]"
        />
        <path
          d={EQUITY_LINE}
          pathLength={1}
          className="stroke-primary [stroke-dasharray:1] motion-safe:animate-[auth-equity-draw_1.4s_ease-out_both]"
          strokeWidth="2"
          strokeLinecap="round"
        />
        {MEMOS.map((m) => (
          <g
            key={m.meta}
            className="motion-safe:animate-[auth-memo-in_500ms_ease-out_both]"
            style={{ animationDelay: `${m.delay}ms` }}
          >
            <circle
              cx={m.marker.cx}
              cy={m.marker.cy}
              r="7"
              className={m.tone === "profit" ? "fill-profit/20" : "fill-loss/20"}
            />
            <circle
              cx={m.marker.cx}
              cy={m.marker.cy}
              r="3"
              className={m.tone === "profit" ? "fill-profit" : "fill-loss"}
            />
          </g>
        ))}
        {/* One focus pass, then a static lens: no idle animation. */}
        <circle cx="540" cy="60" r="42" fill={`url(#${glowId})`} />
        <g data-auth-lens className="motion-safe:animate-[auth-lens-focus_3.6s_linear_both]">
          <circle
            cx="540"
            cy="60"
            r="29"
            fill="#0B132B"
            fillOpacity="0.85"
            stroke="#E5E7EB"
            strokeWidth="3"
          />
          <path d="M561 81 L579 99" stroke="#3B82F6" strokeWidth="5" strokeLinecap="round" />
          <circle cx="540" cy="60" r="4" fill="#3B82F6" />
        </g>
      </svg>

      {MEMOS.map((m) => (
        <div
          key={m.meta}
          style={{ animationDelay: `${m.delay}ms` }}
          className={cn(
            "absolute w-44 rounded-lg border border-border/60 bg-popover/85 px-2.5 py-1.5 text-left shadow-lg backdrop-blur-sm",
            "motion-safe:animate-[auth-memo-in_500ms_ease-out_both]",
            m.chip,
          )}
        >
          <div className="flex items-baseline justify-between gap-2">
            <span className="font-mono text-[10px] tracking-wider text-muted-foreground">
              {m.meta}
            </span>
            <span
              className={cn(
                "font-mono text-[11px] tabular-nums",
                m.tone === "profit" ? "text-profit" : "text-loss",
              )}
            >
              {m.r}
            </span>
          </div>
          <p className="mt-0.5 text-[11px] leading-snug text-popover-foreground">{_(m.note)}</p>
        </div>
      ))}
    </div>
  );
}

/** Split auth frame — brand story panel on the left, quiet form column on the right. */
export function AuthShell({
  children,
  formClassName,
}: {
  children: ReactNode;
  formClassName?: string;
}) {
  const { t: localize } = useSecondaryLingui();

  return (
    <div className="min-h-svh w-full bg-background lg:grid lg:grid-cols-[1.1fr_minmax(0,1fr)]">
      <div className="relative hidden flex-col justify-between gap-10 overflow-hidden bg-sidebar p-10 lg:flex xl:p-14">
        {/* layered washes so the panel reads as lit space, not flat fill */}
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(110%_75%_at_80%_-5%,color-mix(in_oklab,var(--primary)_13%,transparent),transparent_62%)]"
        />
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(90%_55%_at_12%_105%,color-mix(in_oklab,var(--primary)_8%,transparent),transparent_60%)]"
        />

        <div className="relative flex items-center gap-2.5">
          <AppLogo size={48} />
          <p className="text-base font-semibold tracking-tight text-foreground">{BRAND.name}</p>
        </div>

        <div className="relative flex flex-col gap-10">
          <div className="flex max-w-[26rem] flex-col gap-3">
            <h2 className="text-balance text-4xl font-semibold tracking-tight text-foreground xl:text-[2.75rem] xl:leading-[1.1]">
              {BRAND.tagline}
            </h2>
            <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
              {localize({
                id: "auth.yourPrivateTradingJournalDashboardPLCalendarPlaybookAndReportsOn",
                message:
                  "Your private trading journal — dashboard, P&L calendar, playbook, and reports, on your own server.",
              })}
            </p>
          </div>
          <EquityArtwork />
        </div>

        <p className="relative text-xs text-muted-foreground">
          {localize({
            id: "auth.selfHostedYourTradesNeverLeaveYourServer",
            message: "Self-hosted. Your trades never leave your server.",
          })}
        </p>
      </div>

      <main className="relative flex min-h-svh flex-col items-center justify-center px-4 py-10 sm:px-8 lg:min-h-0">
        {/* faint answering glow so the form column reads as part of the same lit space */}
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(65%_45%_at_50%_-8%,color-mix(in_oklab,var(--primary)_6%,transparent),transparent_62%)]"
        />
        <div className="relative mb-8 flex flex-col items-center gap-2.5 lg:hidden">
          <AppLogo size={72} />
          <p className="text-base font-semibold tracking-tight text-foreground">{BRAND.name}</p>
          <p className="text-sm text-muted-foreground">{BRAND.tagline}</p>
        </div>
        <section
          data-auth-surface
          className={cn(
            "relative flex w-full max-w-[22.5rem] min-w-0 flex-col gap-6",
            formClassName,
          )}
        >
          {children}
        </section>
      </main>
    </div>
  );
}
