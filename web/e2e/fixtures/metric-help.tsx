import { createRoot } from "react-dom/client";
import { I18nProvider } from "../../src/i18n";
import { MetricHelp } from "../../src/components/MetricHelp";

createRoot(document.getElementById("root")!).render(
  <I18nProvider>
    <main className="p-6">
      <span>Holding duration</span>
      <MetricHelp metric="holding" />
      <button className="fixed bottom-4 right-4">Outside</button>
    </main>
  </I18nProvider>,
);
