import { useLingui } from "@lingui/react/macro";
import { CircleHelp } from "lucide-react";
import type { KeyboardEvent, ReactNode } from "react";
import { metricDefinitions, type MetricDefinition, type MetricId } from "@/lib/metrics/definitions";
import {
  Popover,
  PopoverTrigger,
  PopoverPopup,
  PopoverTitle,
  PopoverDescription,
  PopoverClose,
} from "@/components/ui/popover";

function stopRowActivation(event: KeyboardEvent) {
  if (event.key === "Enter" || event.key === " ") event.stopPropagation();
}

/** Keep the trigger in the label's intrinsic-width group, even inside stretched cards. */
export function MetricLabel({ children, metric }: { children: ReactNode; metric?: MetricId }) {
  if (!metric) return children;
  return (
    <span
      data-metric-label
      className="inline-flex w-fit min-w-0 max-w-full items-center gap-1 align-middle"
    >
      <span className="min-w-0 wrap-anywhere">{children}</span>
      {metric && <MetricHelp metric={metric} />}
    </span>
  );
}

export function MetricHelp({ metric, breakdown }: { metric: MetricId; breakdown?: ReactNode }) {
  const { i18n, t } = useLingui();
  const entry: MetricDefinition = metricDefinitions[metric];
  const title = i18n._(entry.title);
  return (
    <Popover>
      <PopoverTrigger
        aria-label={t({ id: "metric.help", message: `Explain ${title}` })}
        data-metric-help={metric}
        className="inline-flex size-7 shrink-0 touch-manipulation items-center justify-center rounded-md align-middle text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        onClick={(event) => event.stopPropagation()}
        onKeyDown={stopRowActivation}
      >
        <CircleHelp className="size-3.5" aria-hidden="true" />
      </PopoverTrigger>
      <PopoverPopup
        className="w-88 max-w-[calc(100vw-2rem)] text-left text-sm font-normal normal-case tracking-normal"
        onKeyDown={stopRowActivation}
        onClick={(event) => event.stopPropagation()}
      >
        <PopoverTitle className="pr-6 text-base">{title}</PopoverTitle>
        <PopoverDescription className="mt-3">{i18n._(entry.definition)}</PopoverDescription>
        <dl className="mt-3 space-y-3 break-words">
          <div>
            <dt className="font-medium">{t({ id: "metric.formula", message: "Computation" })}</dt>
            <dd>{i18n._(entry.formula)}</dd>
          </div>
          <div>
            <dt className="font-medium">{t({ id: "metric.unit", message: "Unit" })}</dt>
            <dd>{i18n._(entry.unit)}</dd>
          </div>
          <div>
            <dt className="font-medium">{t({ id: "metric.caveat", message: "Limitations" })}</dt>
            <dd>{i18n._(entry.caveat)}</dd>
          </div>
        </dl>
        {breakdown}
        {entry.documentation && (
          <a className="mt-3 block underline" href={entry.documentation.href}>
            {i18n._(entry.documentation.label)}
          </a>
        )}
        <PopoverClose className="mt-4 rounded-md border px-3 py-1.5 focus-visible:outline-2 focus-visible:outline-ring">
          {t({ id: "metric.close", message: "Close explanation" })}
        </PopoverClose>
      </PopoverPopup>
    </Popover>
  );
}
