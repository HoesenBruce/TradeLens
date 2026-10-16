# Metric help

`MetricHelp` displays a definition, computation/denominator, unit and limitations from
`web/src/lib/metrics/definitions.ts`. IDs are typed and stable. It reads no account data,
performs no finance calculations and writes no application state.

```tsx
import { MetricHelp } from "@/components/MetricHelp";

<span>Holding duration <MetricHelp metric="holding" /></span>
```

Place the control beside a label, outside other buttons/links. In dense tables, use one
control in the column header rather than one per row. The trigger stops click/key
propagation so a help invocation does not open an enclosing row/card.

To register a metric, add a `MetricDefinition` with Lingui `msg` descriptors and a stable ID.
Trace the real API/read-model/formatting paths first; document zero/missing data, units,
denominators, broker and precision limits. Add translations to **en, zh-CN, zh-HK, ja, ko**
using the existing catalogs, then compile. Test every field and interpolation. Existing
`loadLocale` handles English fallback and unsupported locales. Do not clone calculations.

Optional extension seams: `documentation: {href, label}` links to a future knowledge
article; `breakdown?: ReactNode` accepts a future separately validated actual-value view.
Neither actual-value tracing nor a knowledge base is implemented here. Do not pass private
values into general catalog messages.

The shared Base UI Popover supplies dialog semantics, focus management, Escape and
outside-click dismissal. The popup scrolls within available height and fits narrow screens.
Click/tap, Enter and Space invoke it; hover is unnecessary. Close returns focus to the
trigger. Pure explanatory content has no apply/cancel/reset state.

## Evidence and scope (#324)

The sole seed definition is holding duration, checked against
`api/internal/api/trade_holding.go`, `web/src/lib/tradeHolding.ts` and their regression tests.
It distinguishes elapsed minutes from market-local calendar days and unknown boundaries.
Business-screen rollout belongs to #325.

`e2e/fixtures/metric-help.html` is a Vite development test fixture, excluded from the
production entry/bundle. It has no API data dependency. Component and catalog tests cover
keyboard/focus and five locales; `e2e/metric-help.spec.ts` covers 1440/390px, Enter/Space,
Escape, outside click, close, reopen, five-language content and popup bounds. Codex built-in
browser also exercised click, keyboard and narrow-screen rendering.

No backend, accounting, persistence or mobile changes. Mobile apps are outside current
fork acceptance scope. No release/deployment is part of these issues.
