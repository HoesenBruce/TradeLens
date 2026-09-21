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
  const columns: ColumnDef<PerformanceGroup>[] = [
    {
      accessorKey: "key",
      header: "Group",
      cell: ({ row }) => (horizon ? `${row.original.key}D` : row.original.key || "Uncategorized"),
    },
    {
      accessorKey: "source",
      header: "Source",
      cell: ({ row }) => (row.original.source === "ai" ? "AI" : "User"),
    },
    ...(["total", "pending", "validated", "unavailable", "incomplete"] as const).map((key) => ({
      accessorKey: key,
      header: key[0].toUpperCase() + key.slice(1),
    })),
    {
      accessorKey: "hit_rate",
      header: "Directional hit rate",
      cell: ({ row }) => (
        <span className="whitespace-nowrap">
          {row.original.hit_rate === null
            ? "Not available"
            : `${row.original.hit_rate.toFixed(1)}%`}{" "}
          · n={row.original.sample_count} ({row.original.correct} correct)
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
        ← Back to news
      </Button>
      <header>
        <h1 className="text-xl font-semibold">News Thesis performance</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          One sample is one prediction × trading-day horizon. Only finalized results count toward
          hit rates.
        </p>
      </header>
      <section
        aria-label="Performance filters"
        className="grid gap-3 rounded-lg bg-card p-4 sm:grid-cols-4"
      >
        <Field label="Source">
          <NativeSelect
            aria-label="Source"
            value={filters.source ?? ""}
            onChange={(e) => change("source", e.target.value)}
          >
            <NativeSelectOption value="">All sources</NativeSelectOption>
            <NativeSelectOption value="user">User</NativeSelectOption>
            <NativeSelectOption value="ai">AI</NativeSelectOption>
          </NativeSelect>
        </Field>
        <Field label="Asset type">
          <NativeSelect
            aria-label="Asset type"
            value={filters.asset_type ?? ""}
            onChange={(e) => change("asset_type", e.target.value)}
          >
            <NativeSelectOption value="">All assets</NativeSelectOption>
            {["stock", "etf", "index"].map((type) => (
              <NativeSelectOption key={type} value={type}>
                {type.toUpperCase()}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field label="Horizon">
          <NativeSelect
            aria-label="Horizon"
            value={filters.horizon ?? ""}
            onChange={(e) => change("horizon", e.target.value)}
          >
            <NativeSelectOption value="">All horizons</NativeSelectOption>
            {[1, 3, 5, 10, 20].map((h) => (
              <NativeSelectOption key={h} value={String(h)}>
                {h}D
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field label="Symbol">
          <FormInput
            aria-label="Symbol"
            value={filters.symbol ?? ""}
            onChange={(e) => change("symbol", e.target.value)}
          />
        </Field>
        <Field label="Category">
          <FormInput
            aria-label="Category"
            value={filters.category ?? ""}
            onChange={(e) => change("category", e.target.value)}
          />
        </Field>
        <Field label="Published from (UTC)">
          <FormInput
            aria-label="Published from (UTC)"
            type="date"
            value={filters.from ?? ""}
            onChange={(e) => change("from", e.target.value)}
          />
        </Field>
        <Field label="Published to (UTC)">
          <FormInput
            aria-label="Published to (UTC)"
            type="date"
            value={filters.to ?? ""}
            onChange={(e) => change("to", e.target.value)}
          />
        </Field>
        <Button variant="outline" className="self-end" onClick={reset}>
          Reset filters
        </Button>
      </section>
      {query.isLoading ? (
        <Card>
          <div role="status" aria-label="Loading performance">
            <ListSkeleton rows={5} />
          </div>
        </Card>
      ) : query.isError || !report ? (
        <Card>
          <EmptyState
            title="Could not load performance"
            hint={
              query.error instanceof Error
                ? query.error.message
                : "Check the API connection or reset filters."
            }
            actions={<Button onClick={() => void query.refetch()}>Try again</Button>}
          />
        </Card>
      ) : (
        <>
          <section aria-label="Performance counts" className="grid gap-3 sm:grid-cols-5">
            {(["total", "pending", "validated", "unavailable", "incomplete"] as const).map(
              (key) => (
                <StatCard
                  key={key}
                  label={key}
                  value={String(report.counts[key])}
                  hint="Prediction-horizons"
                />
              ),
            )}
          </section>
          <p className="text-sm text-muted-foreground">
            Pending, unavailable and incomplete results are excluded from hit-rate denominators. n
            is the finalized sample count. Small samples are not evidence of reliable accuracy.
            Price direction does not establish news causation.
          </p>
          {report.counts.total === 0 ? (
            <Card>
              <EmptyState
                title="No matching predictions"
                hint="Add predictions or reset filters to see performance."
                actions={
                  <Button variant="outline" onClick={reset}>
                    Reset filters
                  </Button>
                }
              />
            </Card>
          ) : (
            <>
              <Breakdown title="AI / User" rows={report.by_source} />
              <Breakdown title="By horizon" rows={report.by_horizon} horizon />
              <Breakdown title="By asset" rows={report.by_asset} />
              <Breakdown title="By category" rows={report.by_category} />
            </>
          )}
        </>
      )}
    </Page>
  );
}
