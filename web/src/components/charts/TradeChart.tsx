import { useLingui as useLinguiMacro } from "@lingui/react/macro";
import {
  CandlestickSeries,
  ColorType,
  createChart,
  createSeriesMarkers,
  type IChartApi,
  type ISeriesApi,
  type SeriesMarker,
  type Time,
} from "lightweight-charts";
import { ChartCandlestick, Maximize2, Play, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { Execution } from "@/lib/api/types";
import type { MarketBar, BarInterval } from "@/lib/api/market";
import { intlLocale } from "@/lib/locale";
import { cn } from "@/lib/cn";
import { useDisplayPrefs } from "@/lib/displayPrefs";
import { SegmentedControl } from "@/components/SegmentedControl";
import { Skeleton } from "@/components/Skeleton";
import { Button } from "@/components/ui/button";
import { barsToCandlestickData } from "./barsToCandlestickData";
import { utcSecToChartTime } from "./chartTime";
import { BAR_INTERVALS, candleColors, tradeChartTheme } from "./tradeChartTheme";

/** The fill fields the chart draws — synthetic backtest fills qualify too. */
export type ChartFill = Pick<
  Execution,
  "side" | "quantity" | "price" | "executed_at" | "trade_type"
>;

function fillMarkers(fills: ChartFill[], timezone?: string): SeriesMarker<Time>[] {
  return fills.map((f) => ({
    time: utcSecToChartTime(Math.floor(new Date(f.executed_at).getTime() / 1000), timezone),
    position: f.side === "buy" ? "belowBar" : "aboveBar",
    shape: f.side === "buy" ? "arrowUp" : "arrowDown",
    color: f.side === "buy" ? tradeChartTheme.buyMarker : tradeChartTheme.sellMarker,
    text: `${f.quantity} @ ${f.price}`,
  }));
}

export function executionPriceLine(fill: ChartFill) {
  return {
    price: fill.price,
    color: fill.side === "buy" ? tradeChartTheme.buyMarker : tradeChartTheme.sellMarker,
    lineWidth: 1 as const,
    lineStyle: 4 as const,
    axisLabelVisible: true,
    title: `${fill.side === "buy" ? "B" : "S"} ${fill.quantity} · ${fill.trade_type ?? fill.side}`,
  };
}

// Matches the Card header on the trade detail page — the chart is one of its
// card blocks, not a differently-branded panel.
const sectionLabelClass = "text-xs font-medium text-muted-foreground";

export interface TradeChartProps {
  symbol: string;
  bars: MarketBar[] | undefined;
  fills: ChartFill[];
  timezone?: string;
  loading?: boolean;
  error?: boolean;
  errorMessage?: string;
  height?: number;
  targetPrice?: number | null;
  stopPrice?: number | null;
  entryPrice?: number | null;
  interval: BarInterval;
  visibleFrom?: string;
  visibleTo?: string;
  onIntervalChange?: (interval: BarInterval) => void;
  className?: string;
  empty?: boolean;
  /** When true, hide interval chips on empty (non-chartable symbols). */
  hideIntervalWhenEmpty?: boolean;
  /** Opens an expanded modal. Omit when already expanded. */
  onExpand?: () => void;
  /** Hide the CHART label + expand row chrome (modal embeds its own title). */
  hideHeaderLabel?: boolean;
  /**
   * Replay cursor — exclusive end in unix seconds (UTC). Bars opening at or
   * after it are painted transparent and their fill markers hidden, so the
   * time axis stays full-width while candles reveal progressively.
   */
  replayUpTo?: number | null;
  /** Toggles replay mode; renders a play/exit button in the header. */
  onToggleReplay?: () => void;
  replayActive?: boolean;
}

export function TradeChart({
  symbol: _symbol,
  bars,
  fills,
  timezone,
  loading = false,
  error = false,
  errorMessage,
  height = 280,
  targetPrice,
  stopPrice,
  entryPrice,
  interval,
  visibleFrom,
  visibleTo,
  onIntervalChange,
  className,
  empty = false,
  hideIntervalWhenEmpty = false,
  onExpand,
  hideHeaderLabel = false,
  replayUpTo = null,
  onToggleReplay,
  replayActive = false,
}: TradeChartProps) {
  const { t: tr } = useLinguiMacro();

  const locale = intlLocale();
  const priceColorConvention = useDisplayPrefs((s) => s.priceColorConvention);
  const { up: upColor, down: downColor } = candleColors(
    priceColorConvention === "red-up-green-down",
  );
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<"Candlestick"> | null>(null);
  const markersRef = useRef<ReturnType<typeof createSeriesMarkers<Time>> | null>(null);
  const fitKeyRef = useRef<string | null>(null);
  // Price range of the bars revealed so far; null when not replaying.
  const replayRangeRef = useRef<{ lo: number; hi: number } | null>(null);
  const [ready, setReady] = useState(false);

  const showInterval = Boolean(onIntervalChange) && !(hideIntervalWhenEmpty && empty);
  const overlayMessage = loading
    ? null
    : error
      ? (errorMessage ?? tr({ id: "market.chartUnavailable", message: "Chart data unavailable." }))
      : empty
        ? (errorMessage ?? tr({ id: "market.noChart", message: "No chart data." }))
        : bars && bars.length === 0
          ? tr({ id: "market.noBars", message: "No bars for this window." })
          : null;

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const chart = createChart(el, {
      width: el.clientWidth,
      height,
      layout: {
        background: { type: ColorType.Solid, color: tradeChartTheme.background },
        textColor: tradeChartTheme.text,
        fontSize: 11,
      },
      grid: {
        vertLines: { color: tradeChartTheme.grid },
        horzLines: { color: tradeChartTheme.grid },
      },
      rightPriceScale: { borderColor: tradeChartTheme.border },
      timeScale: { borderColor: tradeChartTheme.border, timeVisible: true, secondsVisible: false },
      crosshair: { vertLine: { labelVisible: true }, horzLine: { labelVisible: true } },
    });
    const series = chart.addSeries(CandlestickSeries, {
      upColor,
      downColor,
      borderUpColor: upColor,
      borderDownColor: downColor,
      wickUpColor: upColor,
      wickDownColor: downColor,
      // During replay, scale to the revealed bars only — the full-series
      // default would leak the not-yet-shown price range.
      autoscaleInfoProvider: (
        base: () => { priceRange: { minValue: number; maxValue: number } } | null,
      ) => {
        const r = replayRangeRef.current;
        if (!r) return base();
        return { priceRange: { minValue: r.lo, maxValue: r.hi } };
      },
    });

    chartRef.current = chart;
    seriesRef.current = series;
    fitKeyRef.current = null;
    setReady(true);

    const ro = new ResizeObserver((entries) => {
      const w = entries[0]?.contentRect.width;
      if (w && w > 0) chart.applyOptions({ width: w });
    });
    ro.observe(el);

    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      setReady(false);
    };
  }, [height, upColor, downColor]);

  useEffect(() => {
    chartRef.current?.applyOptions({ localization: { locale } });
  }, [locale, ready]);

  useEffect(() => {
    const series = seriesRef.current;
    const chart = chartRef.current;
    if (!ready || !series || !chart) return;

    if (!bars || bars.length === 0) {
      series.setData([]);
      return;
    }

    let points = barsToCandlestickData(bars, interval, timezone);
    let visibleFills = fills;
    if (replayUpTo != null) {
      // Hide not-yet-reached bars by painting them transparent. Lightweight
      // Charts drops trailing whitespace points, so swapping them for
      // whitespace would collapse the time axis instead of holding it steady.
      const cut = utcSecToChartTime(replayUpTo, timezone) as number;
      const hidden = "rgba(0,0,0,0)";
      points = points.map((p) =>
        "open" in p && (p.time as number) >= cut
          ? { ...p, color: hidden, borderColor: hidden, wickColor: hidden }
          : p,
      );
      visibleFills = fills.filter(
        (f) => Math.floor(new Date(f.executed_at).getTime() / 1000) < replayUpTo,
      );
      let lo = Number.POSITIVE_INFINITY;
      let hi = Number.NEGATIVE_INFINITY;
      for (const b of bars) {
        if (b.time < replayUpTo) {
          lo = Math.min(lo, b.low);
          hi = Math.max(hi, b.high);
        }
      }
      replayRangeRef.current = lo <= hi ? { lo, hi } : null;
    } else {
      replayRangeRef.current = null;
    }
    // The last-value label and price line track the final bar's close, which
    // would spoil the replay ending.
    series.applyOptions({
      lastValueVisible: replayUpTo == null,
      priceLineVisible: replayUpTo == null,
    });
    series.setData(points);

    for (const line of series.priceLines()) {
      series.removePriceLine(line);
    }
    if (entryPrice != null) {
      series.createPriceLine({
        price: entryPrice,
        color: tradeChartTheme.entryLine,
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
        title: tr({ id: "market.entry", message: "Entry" }),
      });
    }
    if (targetPrice != null) {
      series.createPriceLine({
        price: targetPrice,
        color: upColor,
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
        title: tr({ id: "market.target", message: "Target" }),
      });
    }
    if (stopPrice != null) {
      series.createPriceLine({
        price: stopPrice,
        color: downColor,
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
        title: tr({ id: "market.stop", message: "Stop" }),
      });
    }
    for (const fill of visibleFills) {
      series.createPriceLine(executionPriceLine(fill));
    }

    if (visibleFills.length > 0) {
      const markers = fillMarkers(visibleFills, timezone);
      if (markersRef.current) {
        markersRef.current.setMarkers(markers);
      } else {
        markersRef.current = createSeriesMarkers(series, markers);
      }
    } else {
      markersRef.current?.setMarkers([]);
    }

    // Fit only when the bar set changes, not on every replay tick — the
    // whitespaced series keeps the axis full-width, so refitting mid-replay
    // would only clobber the user's zoom.
    const fitKey = `${interval}:${bars.length}:${bars[0]!.time}:${bars.at(-1)!.time}`;
    if (fitKeyRef.current !== fitKey) {
      fitKeyRef.current = fitKey;
      if (visibleFrom && visibleTo) {
        chart.timeScale().setVisibleRange({
          from: utcSecToChartTime(Math.floor(new Date(visibleFrom).getTime() / 1000), timezone),
          to: utcSecToChartTime(Math.floor(new Date(visibleTo).getTime() / 1000), timezone),
        });
      } else {
        chart.timeScale().fitContent();
      }
    }
  }, [
    ready,
    locale,
    tr,
    bars,
    fills,
    interval,
    targetPrice,
    stopPrice,
    entryPrice,
    replayUpTo,
    timezone,
    visibleFrom,
    visibleTo,
    upColor,
    downColor,
  ]);

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        {!hideHeaderLabel && (
          <span className={sectionLabelClass}>{tr({ id: "market.chart", message: "Chart" })}</span>
        )}
        <div
          className={cn("flex items-center gap-1.5", hideHeaderLabel && "w-full justify-between")}
        >
          {showInterval && (
            <SegmentedControl
              value={interval}
              onChange={(v) => onIntervalChange?.(v as BarInterval)}
              options={BAR_INTERVALS}
            />
          )}
          {onToggleReplay && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label={
                replayActive
                  ? tr({ id: "market.exitReplay", message: "Exit replay" })
                  : tr({ id: "market.replayTrade", message: "Replay trade" })
              }
              onClick={onToggleReplay}
            >
              {replayActive ? (
                <X size={14} strokeWidth={1.5} aria-hidden />
              ) : (
                <Play size={14} strokeWidth={1.5} aria-hidden />
              )}
            </Button>
          )}
          {onExpand && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label={tr({ id: "market.expand", message: "Expand chart" })}
              onClick={onExpand}
            >
              <Maximize2 size={14} strokeWidth={1.5} aria-hidden />
            </Button>
          )}
        </div>
      </div>
      <div className="relative overflow-hidden rounded-md bg-muted" style={{ height }}>
        {loading && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-muted/80">
            <Skeleton height={`${height - 24}px`} className="mx-3 w-[calc(100%-1.5rem)]" />
          </div>
        )}
        {!loading && overlayMessage && (
          <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center px-4">
            <div className="flex flex-col items-center gap-2 rounded-md bg-muted px-4 py-8">
              <ChartCandlestick
                size={18}
                strokeWidth={1.5}
                className="text-muted-foreground"
                role="img"
                aria-label={tr({ id: "market.noChartLabel", message: "No chart data" })}
              />
              <p className="m-0 text-center text-xs text-muted-foreground">{overlayMessage}</p>
            </div>
          </div>
        )}
        <div ref={containerRef} className="h-full w-full" />
      </div>
    </div>
  );
}
