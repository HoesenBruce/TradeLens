import { useLingui as useLinguiMacro } from "@lingui/react/macro";
import { Check } from "lucide-react";
import { cn } from "@/lib/cn";

function useImportSteps() {
  const { t: tr } = useLinguiMacro();
  return [
    { step: 1 as const, label: tr({ id: "imports.upload", message: "Upload" }) },
    {
      step: 2 as const,
      labelDefault: tr({ id: "imports.mapColumns", message: "Map columns" }),
      labelJournal: tr({ id: "imports.review", message: "Review" }),
    },
    { step: 3 as const, label: tr({ id: "imports.result", message: "Result" }) },
  ];
}

function stepLabel(item: ReturnType<typeof useImportSteps>[number], format?: string): string {
  if (item.step === 2) {
    return format === "journal_trades" ? item.labelJournal : item.labelDefault;
  }
  return item.label;
}

export function ImportStepIndicator({ current, format }: { current: 1 | 2 | 3; format?: string }) {
  const { t: tr } = useLinguiMacro();
  const STEPS = useImportSteps();
  return (
    <nav aria-label={tr({ id: "imports.progress", message: "Import progress" })}>
      <ol className="flex w-full items-center gap-2 sm:gap-3">
        {STEPS.map((item, index) => {
          const isActive = item.step === current;
          const isDone = item.step < current;
          const label = stepLabel(item, format);

          return (
            <li
              key={item.step}
              className={cn(
                "flex min-w-0 items-center gap-2",
                index < STEPS.length - 1 ? "flex-1" : "shrink-0",
              )}
            >
              <span
                className={cn(
                  "flex size-6 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold tabular-nums",
                  "transition-colors duration-200 ease-out motion-reduce:transition-none",
                  isActive && "bg-primary text-primary-foreground",
                  isDone && "bg-primary/15 text-primary",
                  !isActive && !isDone && "bg-muted text-muted-foreground",
                )}
                aria-current={isActive ? "step" : undefined}
              >
                {isDone ? <Check size={13} strokeWidth={2.5} aria-hidden /> : item.step}
              </span>
              <span
                className={cn(
                  "truncate text-xs",
                  "transition-colors duration-200 ease-out motion-reduce:transition-none",
                  // Narrow screens only have room for the step you're on.
                  isActive ? "inline" : "hidden sm:inline",
                  isActive && "font-medium text-foreground",
                  isDone && "text-muted-foreground",
                  !isActive && !isDone && "text-muted-foreground/60",
                )}
              >
                {label}
              </span>

              {index < STEPS.length - 1 ? (
                <div aria-hidden className="ml-1 min-w-4 flex-1">
                  <div className="relative h-px w-full overflow-hidden rounded-full bg-border">
                    <div
                      className={cn(
                        "absolute inset-y-0 left-0 rounded-full bg-primary",
                        "transition-[width] duration-500 ease-out motion-reduce:transition-none",
                        isDone ? "w-full" : "w-0",
                      )}
                    />
                  </div>
                </div>
              ) : null}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
