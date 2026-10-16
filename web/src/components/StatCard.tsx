import { ResponsiveValue } from "./ResponsiveValue";
import { MetricHelp } from "@/components/MetricHelp";
import type { MetricId } from "@/lib/metrics/definitions";
import { cn } from "@/lib/cn";

interface StatCardProps {
  label: string;
  metric?: MetricId;
  value: string;
  compactValue?: string;
  accent?: "pos" | "neg" | "none";
  hint?: string;
  /**
   * "panel" (default) is a self-contained bordered card, for use in loosely
   * spaced (gap-3+) grids. "bento" is an elevated, borderless cell (one rung
   * up the surface ladder from its typical parent Card) for use in
   * real-gap (gap-3) bento grids, where cells should read as distinct raised
   * compartments instead of blending into the parent panel behind them.
   */
  variant?: "panel" | "bento";
  /**
   * Value type scale. "md" (default) is the original 20px size used by every
   * StatCard outside the Statistics card's tiered layout — do not change its
   * output class, other call sites depend on it staying "text-xl". "lg" and
   * "sm" exist only for that tiered layout's Performance / Behavior & Costs
   * rows.
   */
  size?: "lg" | "md" | "sm";
  /**
   * Content alignment within the cell. "left" (default) preserves the
   * original layout used by every existing call site. "center" centers label
   * and value without forcing a square aspect ratio.
   */
  align?: "left" | "center";
}

function accentColor(accent?: "pos" | "neg" | "none"): string {
  if (accent === "pos") return "var(--profit)";
  if (accent === "neg") return "var(--loss)";
  return "var(--foreground)";
}

function valueSizeClass(size: "lg" | "md" | "sm", centered: boolean): string {
  if (centered) {
    if (size === "lg") return "text-[22px] sm:text-[24px]";
    if (size === "sm") return "text-[14px] sm:text-[15px]";
    return "text-[17px] sm:text-xl";
  }
  if (size === "lg") return "text-[26px]";
  if (size === "sm") return "text-[15px]";
  return "text-xl";
}

export function StatCard({
  label,
  metric,
  value,
  compactValue,
  accent,
  hint,
  variant = "panel",
  size = "md",
  align = "left",
}: StatCardProps) {
  const centered = align === "center";

  return (
    <div
      className={cn(
        "flex min-w-0 max-w-full flex-col wrap-anywhere",
        centered
          ? "w-full items-center gap-2 p-3 text-center"
          : cn("gap-1 px-4", size === "sm" ? "py-2" : "py-3"),
      )}
      style={
        variant === "bento"
          ? {
              background: "var(--muted)",
              borderRadius: "var(--radius)",
            }
          : {
              background: "var(--card)",
              border: "1px solid var(--border)",
              borderRadius: "var(--radius)",
            }
      }
    >
      <span
        className={cn(
          "text-[10px] font-medium tracking-wide uppercase sm:text-xs",
          centered ? "self-center" : undefined,
        )}
        style={{ color: "var(--muted-foreground)" }}
      >
        {label}
        {metric && <MetricHelp metric={metric} />}
      </span>

      <span
        className={cn(
          valueSizeClass(size, centered),
          "font-semibold leading-none tabular-nums",
          "w-full min-w-0 max-w-full wrap-anywhere",
        )}
        style={{
          color: accentColor(accent),
          transition: "color 150ms",
        }}
      >
        {compactValue != null ? (
          <ResponsiveValue value={value} compactValue={compactValue} />
        ) : (
          value
        )}
      </span>
      {hint ? (
        <span
          className={cn("text-xs", centered && "max-w-full text-[10px] leading-snug")}
          style={{ color: "var(--muted-foreground)" }}
        >
          {hint}
        </span>
      ) : null}
    </div>
  );
}
