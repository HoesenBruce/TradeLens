import type { AccountValue, AccountValuePoint } from "./api/types";

const statusPriority: Record<string, number> = {
  complete: 0,
  incomplete_missing_price: 1,
  incomplete_invalid_sequence: 2,
  unsupported_corporate_action: 3,
};

/** Convert each API series to one currency before adding any monetary fields. */
export function combineAccountValues(
  series: AccountValue[],
  currency: string,
  rates: Record<string, number | undefined>,
): AccountValue {
  const dates = [
    ...new Set(series.flatMap((account) => account.points.map((point) => point.date))),
  ].sort();
  const accounts = series.map((account) => {
    const rate = account.currency === currency ? 1 : rates[account.currency];
    if (rate == null || !Number.isFinite(rate) || rate <= 0) {
      throw new Error(`Missing FX rate: ${account.currency}/${currency}`);
    }
    return { rate, points: new Map(account.points.map((point) => [point.date, point])) };
  });
  return {
    currency,
    timezone: series[0]?.timezone ?? "Asia/Tokyo",
    adjustment_status: series[0]?.adjustment_status ?? "unadjusted",
    points: dates.map((date) => {
      const combined: AccountValuePoint = {
        date,
        estimated_account_value: 0,
        contributed_capital: 0,
        cash_balance: 0,
        open_position_value: 0,
        realized_pnl: 0,
        unrealized_pnl: 0,
        status: "complete",
        warnings: [],
      };
      for (const account of accounts) {
        const point = account.points.get(date);
        if (!point) {
          // Dates before an account's history contribute zero; gaps must not show a partial total.
          const firstDate = account.points.keys().next().value;
          if (firstDate && date >= firstDate) {
            combined.estimated_account_value =
              combined.open_position_value =
              combined.unrealized_pnl =
                null;
            if (combined.status === "complete") combined.status = "incomplete_missing_price";
          }
          continue;
        }
        for (const key of ["contributed_capital", "cash_balance", "realized_pnl"] as const) {
          combined[key] += point[key] * account.rate;
        }
        for (const key of [
          "estimated_account_value",
          "open_position_value",
          "unrealized_pnl",
        ] as const) {
          combined[key] =
            combined[key] == null || point[key] == null
              ? null
              : combined[key] + point[key] * account.rate;
        }
        if ((statusPriority[point.status] ?? 1) > (statusPriority[combined.status] ?? 1)) {
          combined.status = point.status;
        }
        combined.warnings!.push(...(point.warnings ?? []));
      }
      return combined;
    }),
  };
}
