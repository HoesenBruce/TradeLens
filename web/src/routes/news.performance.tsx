import { fmtPct } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { useLingui } from "@lingui/react/macro";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Card } from "@/components/Card";
import { DataTable } from "@/components/DataTable";
import { EmptyState } from "@/components/EmptyState";
import { Field } from "@/components/Field";
import { FormInput } from "@/components/FormInput";
import { Page } from "@/components/Page";
import { StatCard } from "@/components/StatCard";
import { ListSkeleton } from "@/components/skeletons/list-skeleton";
import { Button } from "@/components/ui/button";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import {
  getNewsPerformance,
  performanceFilterKeys,
  type PerformanceFilters,
  type PerformanceGroup,
} from "@/lib/api/newsPerformance";
import type { ColumnDef } from "@/lib/table";

export const Route = createFileRoute("/news/performance")({
  validateSearch: (search: Record<string, unknown>): PerformanceFilters =>
    Object.fromEntries(
      performanceFilterKeys.flatMap((key) =>
        typeof search[key] === "string" && search[key] ? [[key, search[key]]] : [],
      ),
    ),
  component: NewsPerformancePage,
});

function Breakdown({
  title,
  rows,
  horizon = false,
}: {
  title: string;
  rows: PerformanceGroup[];
  horizon?: boolean;
}) {
  const { t } = useLingui();
  const labels = {
    total: t({ id: "news.total", message: "Total" }),
    pending: t({ id: "news.pending", message: "Pending" }),
    validated: t({ id: "news.validated", message: "Validated" }),
    unavailable: t({ id: "news.unavailable", message: "Unavailable" }),
    incomplete: t({ id: "news.incomplete", message: "Incomplete" }),
    stock: t({ id: "news.stock", message: "Stock" }),
    etf: "ETF",
    index: t({ id: "news.index", message: "Index" }),
  };
  const columns: ColumnDef<PerformanceGroup>[] = [
    {
      accessorKey: "key",
      header: t({ id: "news.group", message: "Group" }),
      cell: ({ row }) =>
        horizon
          ? t({ id: "news.groupDays", message: `${row.original.key}D` })
          : row.original.key || t({ id: "news.uncategorized", message: "Uncategorized" }),
    },
    {
      accessorKey: "source",
      header: t({ id: "news.source", message: "Source" }),
      cell: ({ row }) =>
        row.original.source === "ai" ? "AI" : t({ id: "news.user", message: "User" }),
    },
    ...(["total", "pending", "validated", "unavailable", "incomplete"] as const).map((key) => ({
      accessorKey: key,
      header: labels[key],
    })),
    {
      accessorKey: "hit_rate",
      header: t({ id: "news.hitRate", message: "Directional hit rate" }),
      cell: ({ row }) => (
        <span className="whitespace-nowrap">
          {row.original.hit_rate === null
            ? t({ id: "news.notAvailable", message: "Not available" })
            : fmtPct(row.original.hit_rate / 100, intlLocale(), 1)}{" "}
          {t({
            id: "news.sampleCount",
            message: `· n=${row.original.sample_count} (${row.original.correct} correct)`,
          })}
        </span>
      ),
    },
  ];
  return (
    <section aria-label={title}>
      <Card title={title}>
        <DataTable columns={columns} data={rows} />
      </Card>
    </section>
  );
}

function NewsPerformancePage() {
  const { t } = useLingui();
  const labels = {
    total: t({ id: "news.total", message: "Total" }),
    pending: t({ id: "news.pending", message: "Pending" }),
    validated: t({ id: "news.validated", message: "Validated" }),
    unavailable: t({ id: "news.unavailable", message: "Unavailable" }),
    incomplete: t({ id: "news.incomplete", message: "Incomplete" }),
    stock: t({ id: "news.stock", message: "Stock" }),
    etf: "ETF",
    index: t({ id: "news.index", message: "Index" }),
  };
  const filters = Route.useSearch();
  const navigate = Route.useNavigate();
  const query = useQuery({
    queryKey: ["news-performance", filters],
    queryFn: () => getNewsPerformance(filters),
  });
  const change = (key: keyof PerformanceFilters, value: string) =>
    void navigate({
      search: (previous) => ({ ...previous, [key]: value || undefined }),
      replace: true,
    });
  const reset = () => void navigate({ search: {}, replace: true });
  const report = query.data;
  return (
    <Page>
      <Button variant="ghost" className="self-start" render={<Link to="/news" />}>
        {t({ id: "news.back", message: "← Back to news" })}
      </Button>
      <header>
        <h1 className="text-xl font-semibold">
          {t({ id: "news.performanceTitle", message: "News Thesis performance" })}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t({
            id: "news.sampleHint",
            message:
              "One sample is one prediction × trading-day horizon. Only finalized results count toward hit rates.",
          })}
        </p>
      </header>
      <section
        aria-label={t({ id: "news.performanceFilters", message: "Performance filters" })}
        className="grid gap-3 rounded-lg bg-card p-4 sm:grid-cols-4"
      >
        <Field label={t({ id: "news.source", message: "Source" })}>
          <NativeSelect
            aria-label={t({ id: "news.source", message: "Source" })}
            value={filters.source ?? ""}
            onChange={(e) => change("source", e.target.value)}
          >
            <NativeSelectOption value="">
              {t({ id: "news.allSources", message: "All sources" })}
            </NativeSelectOption>
            <NativeSelectOption value="user">
              {t({ id: "news.user", message: "User" })}
            </NativeSelectOption>
            <NativeSelectOption value="ai">AI</NativeSelectOption>
          </NativeSelect>
        </Field>
        <Field label={t({ id: "news.assetType", message: "Asset type" })}>
          <NativeSelect
            aria-label={t({ id: "news.assetType", message: "Asset type" })}
            value={filters.asset_type ?? ""}
            onChange={(e) => change("asset_type", e.target.value)}
          >
            <NativeSelectOption value="">
              {t({ id: "news.allAssets", message: "All assets" })}
            </NativeSelectOption>
            {["stock", "etf", "index"].map((type) => (
              <NativeSelectOption key={type} value={type}>
                {labels[type as keyof typeof labels]}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field label={t({ id: "news.horizon", message: "Horizon" })}>
          <NativeSelect
            aria-label={t({ id: "news.horizon", message: "Horizon" })}
            value={filters.horizon ?? ""}
            onChange={(e) => change("horizon", e.target.value)}
          >
            <NativeSelectOption value="">
              {t({ id: "news.allHorizons", message: "All horizons" })}
            </NativeSelectOption>
            {[1, 3, 5, 10, 20].map((h) => (
              <NativeSelectOption key={h} value={String(h)}>
                {t({ id: "news.days", message: `${h}D` })}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field label={t({ id: "news.symbol", message: "Symbol" })}>
          <FormInput
            aria-label={t({ id: "news.symbol", message: "Symbol" })}
            value={filters.symbol ?? ""}
            onChange={(e) => change("symbol", e.target.value)}
          />
        </Field>
        <Field label={t({ id: "news.category", message: "Category" })}>
          <FormInput
            aria-label={t({ id: "news.category", message: "Category" })}
            value={filters.category ?? ""}
            onChange={(e) => change("category", e.target.value)}
          />
        </Field>
        <Field label={t({ id: "news.fromUtc", message: "Published from (UTC)" })}>
          <FormInput
            aria-label={t({ id: "news.fromUtc", message: "Published from (UTC)" })}
            type="date"
            value={filters.from ?? ""}
            onChange={(e) => change("from", e.target.value)}
          />
        </Field>
        <Field label={t({ id: "news.toUtc", message: "Published to (UTC)" })}>
          <FormInput
            aria-label={t({ id: "news.toUtc", message: "Published to (UTC)" })}
            type="date"
            value={filters.to ?? ""}
            onChange={(e) => change("to", e.target.value)}
          />
        </Field>
        <Button variant="outline" className="self-end" onClick={reset}>
          {t({ id: "news.resetFilters", message: "Reset filters" })}
        </Button>
      </section>
      {query.isLoading ? (
        <Card>
          <div
            role="status"
            aria-label={t({ id: "news.loadingPerformance", message: "Loading performance" })}
          >
            <ListSkeleton rows={5} />
          </div>
        </Card>
      ) : query.isError || !report ? (
        <Card>
          <EmptyState
            title={t({ id: "news.performanceFailed", message: "Could not load performance" })}
            hint={
              query.error instanceof Error
                ? query.error.message
                : t({
                    id: "news.performanceRetryHint",
                    message: "Check the API connection or reset filters.",
                  })
            }
            actions={
              <Button onClick={() => void query.refetch()}>
                {t({ id: "news.retry", message: "Try again" })}
              </Button>
            }
          />
        </Card>
      ) : (
        <>
          <section
            aria-label={t({ id: "news.performanceCounts", message: "Performance counts" })}
            className="grid gap-3 sm:grid-cols-5"
          >
            {(["total", "pending", "validated", "unavailable", "incomplete"] as const).map(
              (key) => (
                <StatCard
                  key={key}
                  label={labels[key]}
                  value={String(report.counts[key])}
                  hint={t({ id: "news.predictionHorizons", message: "Prediction-horizons" })}
                />
              ),
            )}
          </section>
          <p className="text-sm text-muted-foreground">
            {t({
              id: "news.statisticsWarning",
              message:
                "Pending, unavailable and incomplete results are excluded from hit-rate denominators. n is the finalized sample count. Small samples are not evidence of reliable accuracy. Price direction does not establish news causation.",
            })}
          </p>
          {report.counts.total === 0 ? (
            <Card>
              <EmptyState
                title={t({ id: "news.noMatchingPredictions", message: "No matching predictions" })}
                hint={t({
                  id: "news.performanceEmptyHint",
                  message: "Add predictions or reset filters to see performance.",
                })}
                actions={
                  <Button variant="outline" onClick={reset}>
                    {t({ id: "news.resetFilters", message: "Reset filters" })}
                  </Button>
                }
              />
            </Card>
          ) : (
            <>
              <Breakdown
                title={t({ id: "news.aiUser", message: "AI / User" })}
                rows={report.by_source.map((row) => ({
                  ...row,
                  key: row.source === "ai" ? "AI" : t({ id: "news.user", message: "User" }),
                }))}
              />
              <Breakdown
                title={t({ id: "news.byHorizon", message: "By horizon" })}
                rows={report.by_horizon}
                horizon
              />
              <Breakdown
                title={t({ id: "news.byAsset", message: "By asset" })}
                rows={report.by_asset.map((row) => ({
                  ...row,
                  key: row.key.replace(
                    /^(stock|etf|index)(?= \/ )/,
                    (type) => labels[type as "stock" | "etf" | "index"],
                  ),
                }))}
              />
              <Breakdown
                title={t({ id: "news.byCategory", message: "By category" })}
                rows={report.by_category}
              />
            </>
          )}
        </>
      )}
    </Page>
  );
}
