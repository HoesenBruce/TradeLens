import { useLayoutEffect, useRef, useState } from "react";
import { cn } from "@/lib/cn";
import { Tooltip, TooltipPopup, TooltipTrigger } from "./ui/tooltip";

/** Measure the full text independently so abbreviating never changes the fit decision. */
export function ResponsiveMoney({
  value,
  compactValue,
  className,
}: {
  value: string;
  compactValue: string;
  className?: string;
}) {
  const container = useRef<HTMLSpanElement>(null);
  const fullText = useRef<HTMLSpanElement>(null);
  const [compact, setCompact] = useState(false);

  useLayoutEffect(() => {
    const el = container.current;
    const text = fullText.current;
    if (!el || !text) return;
    const measure = () => setCompact(text.getBoundingClientRect().width + 4 > el.clientWidth);
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    observer.observe(text);
    return () => observer.disconnect();
  }, [value]);

  return (
    <Tooltip>
      <TooltipTrigger
        render={<span ref={container} />}
        disabled={!compact}
        tabIndex={compact ? 0 : undefined}
        aria-label={compact ? value : undefined}
        data-money-format={compact ? "compact" : "full"}
        className={cn(
          "grid w-full min-w-0 max-w-full grid-cols-[minmax(0,1fr)] rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
          className,
        )}
      >
        <span
          aria-hidden="true"
          className="pointer-events-none invisible col-start-1 row-start-1 overflow-hidden"
        >
          <span ref={fullText} className="block w-max whitespace-nowrap">
            {value}
          </span>
        </span>
        <span className="col-start-1 row-start-1 truncate">{compact ? compactValue : value}</span>
      </TooltipTrigger>
      {compact && (
        <TooltipPopup className="max-w-[calc(100vw-2rem)] break-all">{value}</TooltipPopup>
      )}
    </Tooltip>
  );
}
