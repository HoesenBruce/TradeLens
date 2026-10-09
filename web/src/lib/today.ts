import { useEffect, useState } from "react";
import { dayKeyInTz } from "./calendar";
import { resolveMarketTimezone, useDisplayPrefs } from "./displayPrefs";

/** Refresh while Home stays open across market midnight. */
export function useMarketToday(): string {
  const preference = useDisplayPrefs((state) => state.marketTimezone);
  const [today, setToday] = useState(() =>
    dayKeyInTz(new Date().toISOString(), resolveMarketTimezone(preference)),
  );
  useEffect(() => {
    const tick = () =>
      setToday(dayKeyInTz(new Date().toISOString(), resolveMarketTimezone(preference)));
    tick();
    const timer = window.setInterval(tick, 1000);
    window.addEventListener("focus", tick);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("focus", tick);
    };
  }, [preference]);
  return today;
}
