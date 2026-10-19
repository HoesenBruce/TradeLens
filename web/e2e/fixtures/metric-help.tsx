import { createRoot } from "react-dom/client";
import { I18nProvider, loadLocale } from "../../src/i18n";
import { useLingui } from "@lingui/react";
import { MetricLabel } from "../../src/components/MetricHelp";
import { StatCard } from "../../src/components/StatCard";
import { StatCell } from "../../src/components/StatCell";
import { metricDefinitions } from "../../src/lib/metrics/definitions";

function Fixture() {
  const { i18n } = useLingui();
  const label = i18n._(metricDefinitions.holding.title);
  const longLabel = i18n._(metricDefinitions.quantities.title);
  return (
    <main className="space-y-6 p-6">
      <label>
        Locale{" "}
        <select
          aria-label="Locale"
          value={i18n.locale}
          onChange={(event) => void loadLocale(event.target.value)}
        >
          {["en", "zh-CN", "zh-HK", "ja", "ko"].map((locale) => (
            <option key={locale}>{locale}</option>
          ))}
        </select>
      </label>
      <div data-testid="short-label" className="flex justify-between rounded-lg border p-4">
        <MetricLabel metric="holding">{label}</MetricLabel>
        <span>42</span>
      </div>
      <div data-testid="long-label" className="w-48 max-w-full">
        <StatCell label={longLabel} metric="quantities">
          100 / 80 / 20
        </StatCell>
      </div>
      <div data-testid="card" className="max-w-full">
        <StatCard label={label} metric="holding" value="42" />
      </div>
      <div data-testid="no-help">
        <StatCard label="No help" value="42" />
      </div>
      <button className="fixed bottom-4 right-4">Outside</button>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <I18nProvider>
    <Fixture />
  </I18nProvider>,
);
