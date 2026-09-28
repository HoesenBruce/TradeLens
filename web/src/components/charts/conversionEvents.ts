import type { Execution } from "@/lib/api/types";

type ConversionFill = Pick<Execution, "side" | "quantity" | "executed_at"> &
  Partial<Pick<Execution, "details">>;

/** Only explicit conversion metadata qualifies; matching ordinary fills never do. */
export function isMarginToCash(fill: Partial<Pick<Execution, "side" | "details">>): boolean {
  const d = fill.details;
  return Boolean(
    d?.event_type === "position_conversion" &&
    (d.conversion_type === "genbiki" || d.conversion_type === "margin_to_cash") &&
    d.conversion_id &&
    ((d.lot === "sbi:margin-long" && fill.side === "sell") ||
      (d.lot === "sbi:cash" && fill.side === "buy")),
  );
}

/** Trade details normally contain one leg because cash and margin trades are separate. */
export function conversionEvents<T extends ConversionFill>(fills: T[]) {
  const groups = new Map<string, T[]>();
  for (const fill of fills) {
    if (!isMarginToCash(fill)) continue;
    const id = fill.details!.conversion_id!;
    groups.set(id, [...(groups.get(id) ?? []), fill]);
  }
  const valid = new Set<string>();
  for (const [id, legs] of groups) {
    if (
      legs.length === 1 ||
      (legs.length === 2 &&
        legs[0]!.side !== legs[1]!.side &&
        legs[0]!.quantity === legs[1]!.quantity &&
        legs[0]!.executed_at === legs[1]!.executed_at)
    )
      valid.add(id);
  }
  const seen = new Set<string>();
  return fills.flatMap((fill) => {
    const id = isMarginToCash(fill) ? fill.details!.conversion_id! : undefined;
    if (!id || !valid.has(id)) return [{ fill, conversion: false, legs: [fill] }];
    if (seen.has(id)) return [];
    seen.add(id);
    return [{ fill, conversion: true, legs: groups.get(id)! }];
  });
}
