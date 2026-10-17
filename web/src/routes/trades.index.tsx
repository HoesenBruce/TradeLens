import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { TradesView } from "@/app/screens/TradesView";
import { TradeDetailSheet } from "@/components/TradeDetailSheet";
import { accountBaseCurrency } from "@/lib/displayPrefs";
import { useFilterParams, useFilters } from "@/lib/filters";
import { useAccounts } from "@/lib/hooks/useAccounts";
import { useTrades } from "@/lib/hooks/useTrades";
import {
  buildMarketFacetOptions,
  buildSymbolFacetOptions,
  buildTagFacetOptions,
  applyTradeFilters,
  tradePageFilters,
  type TradeFilter,
} from "@/lib/tradeFilters";
import { useUI } from "@/lib/ui";

export const Route = createFileRoute("/trades/")({
  component: TradesPage,
});

function TradesPage() {
  const filters = useFilterParams();
  const accountIds = useFilters((s) => s.accountIds);
  const from = useFilters((s) => s.from);
  const to = useFilters((s) => s.to);
  const symbols = useFilters((s) => s.symbols);
  const tradeStatus = useFilters((s) => s.tradeStatus);
  const tradeConditions = useFilters((s) => s.tradeConditions);
  const tagIds = useFilters((s) => s.tagIds);
  const markets = useFilters((s) => s.markets);
  const conditions = useMemo(
    () => tradePageFilters({ symbols, tradeStatus, tagIds, markets, tradeConditions }),
    [symbols, tradeStatus, tagIds, markets, tradeConditions],
  );
  const setSymbols = useFilters((s) => s.setSymbols);
  const setRange = useFilters((s) => s.setRange);
  const setTradeStatus = useFilters((s) => s.setTradeStatus);
  const setTagIds = useFilters((s) => s.setTagIds);
  const setMarkets = useFilters((s) => s.setMarkets);
  const navigate = useNavigate();
  const openModal = useUI((s) => s.openModal);
  const [selectedTradeId, setSelectedTradeId] = useState<string | null>(null);

  const tradesQ = useTrades({ ...filters, symbol: undefined });
  const accountsQ = useAccounts();
  const accounts = accountsQ.data ?? [];
  const currency = tradesQ.currency ?? accountBaseCurrency(accounts, accountIds);

  const hasNarrowingFilters = !!(from || to || conditions.length);
  const scopeFilters = accountIds?.length ? { account_id: accountIds.join(",") } : {};
  const scopeQ = useTrades(scopeFilters);

  const rawTrades = tradesQ.data;
  const tagOptions = useMemo(() => buildTagFacetOptions(rawTrades ?? []), [rawTrades]);
  const marketOptions = useMemo(() => buildMarketFacetOptions(rawTrades ?? []), [rawTrades]);
  const symbolOptions = useMemo(() => buildSymbolFacetOptions(scopeQ.data ?? []), [scopeQ.data]);
  const trades = applyTradeFilters(rawTrades ?? [], conditions);

  function handleConditionsChange(next: TradeFilter[]) {
    // Shared scope supports inclusion only; keep exclusion operators local to Trades.
    const values = (field: string) => {
      const filter = next.find((f) => f.field === field);
      return filter?.operator === "is_not_any_of" || filter?.operator === "includes_all"
        ? undefined
        : filter?.values;
    };
    useFilters.setState({
      tradeConditions: next,
      symbols: values("symbol"),
      markets: values("market"),
      tagIds: values("tags"),
      tradeStatus: undefined,
    });
  }

  function handleClearFilters() {
    useFilters.setState({ tradeConditions: undefined });
    setSymbols(undefined);
    setRange(undefined, undefined);
    setTradeStatus(undefined);
    setTagIds(undefined);
    setMarkets(undefined);
  }

  return (
    <>
      <TradesView
        trades={trades}
        loading={tradesQ.isLoading}
        error={tradesQ.isError}
        currency={currency}
        symbols={symbols ?? []}
        onSymbolsChange={setSymbols}
        onSelectTrade={(id) => setSelectedTradeId(id)}
        onOpenFullPage={(id) => void navigate({ to: "/trades/$id", params: { id } })}
        onDeleted={(id) => {
          if (selectedTradeId === id) setSelectedTradeId(null);
        }}
        totalInScope={scopeQ.data?.length ?? 0}
        scopeLoading={scopeQ.isLoading}
        hasNarrowingFilters={hasNarrowingFilters}
        conditions={conditions}
        onConditionsChange={handleConditionsChange}
        tagIds={tagIds}
        tagOptions={tagOptions}
        markets={markets}
        marketOptions={marketOptions}
        symbolOptions={symbolOptions}
        onClearFilters={handleClearFilters}
        onImport={() => navigate({ to: "/import" })}
        onNewTrade={() => openModal("new-trade")}
        onRetry={() => void tradesQ.refetch()}
      />
      <TradeDetailSheet tradeId={selectedTradeId} onClose={() => setSelectedTradeId(null)} />
    </>
  );
}
