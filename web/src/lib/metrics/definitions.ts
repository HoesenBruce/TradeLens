import type { MessageDescriptor } from "@lingui/core";
import { msg } from "@lingui/core/macro";

export interface MetricDefinition {
  title: MessageDescriptor;
  definition: MessageDescriptor;
  formula: MessageDescriptor;
  unit: MessageDescriptor;
  caveat: MessageDescriptor;
  documentation?: { href: string; label: MessageDescriptor };
}

// Stable IDs identify explanations, never computations or user data.
export const metricDefinitions = {
  holding: {
    title: msg({ id: "metric.holding.title", message: "Holding duration" }),
    definition: msg({
      id: "metric.holding.definition",
      message: "Time from the recorded opening to the close, or to now for an open trade.",
    }),
    formula: msg({
      id: "metric.holding.formula",
      message:
        "Precise endpoints: elapsed time, rounded to minutes for display (under one minute: <1m). Any date-only endpoint: difference between exchange-local calendar dates; same date = 0 days, next date = 1 day.",
    }),
    unit: msg({
      id: "metric.holding.unit",
      message: "Days, hours and minutes; calendar days for date-only records.",
    }),
    caveat: msg({
      id: "metric.holding.caveat",
      message:
        "Date-only SBI or CSV fills do not establish intraday holding time. Missing opening history, unknown precision or invalid boundaries show —. The API supplies the market timezone and precision; the viewer timezone does not change calendar days.",
    }),
  },
} satisfies Record<string, MetricDefinition>;

export type MetricId = keyof typeof metricDefinitions;
