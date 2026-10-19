import { getIntlLocale } from "@/lib/locale";
import { useLingui } from "@lingui/react";
import { useMoneyFormatters } from "@/lib/useMoneyFormatters";
import { useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { cn } from "@/lib/cn";
import { Tooltip, TooltipPopup, TooltipTrigger } from "./ui/tooltip";

/** Measure the full text independently so abbreviating never changes the fit decision. */
export function ResponsiveValue({
  value,
  compactValue = value,
  className,
  disclosure = true,
  renderValue,
}: {
  value: string;
  /** Preserve the existing inline value slot in Lingui Trans messages. */
  children?: ReactNode;
  compactValue?: string;
  disclosure?: boolean;
  renderValue?: (value: string) => ReactNode;
  className?: string;
}) {
  const container = useRef<HTMLSpanElement>(null);
  const fullText = useRef<HTMLSpanElement>(null);
  const compactText = useRef<HTMLSpanElement>(null);
  const [compact, setCompact] = useState(false);
  const [compactSize, setCompactSize] = useState<number>();
  const [open, setOpen] = useState(false);

  useLayoutEffect(() => {
    const el = container.current;
    const text = fullText.current;
    if (!el || !text) return;
    const measure = () => {
      if (el.clientWidth <= 0) return;
      const needsCompact = text.getBoundingClientRect().width + 4 > el.clientWidth;
      setCompact(needsCompact);
      const fontSize = Number.parseFloat(getComputedStyle(el).fontSize) || 16;
      const compactWidth = compactText.current?.getBoundingClientRect().width ?? 0;
      const size =
        compactWidth > 0
          ? Math.min(
              fontSize,
              Math.max(Math.min(fontSize, 18), (fontSize * (el.clientWidth - 4)) / compactWidth),
            )
          : fontSize;
      setCompactSize(needsCompact && size < fontSize ? size : undefined);
      if (!needsCompact) setOpen(false);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    observer.observe(text);
    if (compactText.current) observer.observe(compactText.current);
    return () => observer.disconnect();
  }, [value, compactValue]);

  return (
    <Tooltip open={open && compact && disclosure} onOpenChange={setOpen}>
      <TooltipTrigger
        render={<span ref={container} />}
        disabled={!compact || !disclosure}
        closeOnClick={false}
        onClick={
          disclosure && compact
            ? (event) => {
                event.stopPropagation();
                setOpen(true);
              }
            : undefined
        }
        onFocus={compact && disclosure ? () => setOpen(true) : undefined}
        tabIndex={compact && disclosure ? 0 : undefined}
        aria-label={compact ? value : undefined}
        role={compact && disclosure ? "button" : undefined}
        aria-expanded={compact && disclosure ? open : undefined}
        onKeyDown={
          compact && disclosure
            ? (event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  event.stopPropagation();
                  setOpen((v) => !v);
                }
              }
            : undefined
        }
        data-money-format={compact ? "compact" : "full"}
        className={cn(
          "grid w-full min-w-0 max-w-full grid-cols-[minmax(0,1fr)] rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
          className,
        )}
      >
        <span
          aria-hidden="true"
          className="pointer-events-none invisible col-start-1 row-start-1 grid overflow-hidden"
        >
          <span
            ref={fullText}
            data-value={value}
            className="col-start-1 row-start-1 block w-max whitespace-nowrap before:content-[attr(data-value)]"
          />
          <span
            ref={compactText}
            data-value={compactValue}
            className="col-start-1 row-start-1 block w-max whitespace-nowrap before:content-[attr(data-value)]"
          />
        </span>
        <span className="col-start-1 row-start-1 truncate" style={{ fontSize: compactSize }}>
          {renderValue
            ? renderValue(compact ? compactValue : value)
            : compact
              ? compactValue
              : value}
        </span>
      </TooltipTrigger>
      {compact && disclosure && (
        <TooltipPopup role="tooltip" className="max-w-[calc(100vw-2rem)] break-all">
          {value}
        </TooltipPopup>
      )}
    </Tooltip>
  );
}

export function Money({
  value,
  currency,
  locale,
  signed = false,
  disclosure = true,
}: {
  value: number;
  currency: string;
  locale?: string;
  signed?: boolean;
  disclosure?: boolean;
}) {
  const { i18n } = useLingui();
  const displayLocale = locale ?? getIntlLocale(i18n.locale);
  const { fmtMoney, fmtMoneyCompact, fmtSignedMoney, fmtSignedMoneyCompact } = useMoneyFormatters();
  return (
    <ResponsiveValue
      disclosure={disclosure}
      value={(signed ? fmtSignedMoney : fmtMoney)(value, currency, displayLocale)}
      compactValue={(signed ? fmtSignedMoneyCompact : fmtMoneyCompact)(
        value,
        currency,
        displayLocale,
      )}
    />
  );
}
