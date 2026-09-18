import type { CandlestickData, UTCTimestamp } from "lightweight-charts";
import type { BarInterval, MarketBar } from "@/lib/api/market";
import { CHART_TIME_ZONE, utcSecToChartTime } from "./chartTime";

export const INTERVAL_SEC: Record<BarInterval, number> = {
  "1": 60,
  "5": 300,
  "15": 900,
  "60": 3600,
  "240": 14_400,
  D: 86_400,
};

/**
 * Map returned market bars to lightweight-charts candlestick data. Times are
 * shifted to {@link CHART_TIME_ZONE} because LWC labels UTCTimestamps as UTC
 * wall-clock.
 */
export function barsToCandlestickData(
  bars: MarketBar[],
  _interval: BarInterval,
  timeZone: string = CHART_TIME_ZONE,
): CandlestickData<UTCTimestamp>[] {
  const toTime = (utcSec: number) => utcSecToChartTime(utcSec, timeZone);
  return bars.map((bar) => ({
    time: toTime(bar.time),
    open: bar.open,
    high: bar.high,
    low: bar.low,
    close: bar.close,
  }));
}
