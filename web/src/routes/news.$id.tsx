import { Link, createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { NewsFormDialog } from "@/app/screens/NewsView";
import { Card } from "@/components/Card";
import { EmptyState } from "@/components/EmptyState";
import { NewsAnalysisReview } from "@/components/NewsAnalysisReview";
import { NewsPredictions } from "@/components/NewsPredictions";
import { Page } from "@/components/Page";
import { Pill } from "@/components/Pill";
import { ListSkeleton } from "@/components/skeletons/list-skeleton";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";
import { fmtDateTime } from "@/lib/format";
import { useNewsDetail, usePredictionActions, useUpdateNews } from "@/lib/hooks/useNews";

export const Route = createFileRoute("/news/$id")({ component: NewsDetailPage });

function NewsDetailPage() {
  const { id } = Route.useParams();
  const query = useNewsDetail(id);
  const update = useUpdateNews();
  const predictionActions = usePredictionActions();
  const [editing, setEditing] = useState(false);
  const news = query.data;
  const missing = query.error instanceof ApiError && query.error.status === 404;
  return (
    <Page>
      <Button variant="ghost" className="self-start" render={<Link to="/news" />}>
        ← Back to news
      </Button>
      {query.isLoading ? (
        <Card>
          <ListSkeleton rows={5} />
        </Card>
      ) : query.isError || !news ? (
        <Card>
          <EmptyState
            title={missing ? "News thesis not found" : "Could not load news thesis"}
            hint={
              missing
                ? "This entry may have been deleted or is not available to your account."
                : "Check the API connection and try again."
            }
            actions={
              !missing ? (
                <Button variant="outline" onClick={() => void query.refetch()}>
                  Try again
                </Button>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <>
          <header className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h1 className="text-xl font-semibold">{news.title}</h1>
              <p className="mt-1 text-sm text-muted-foreground">
                {news.source} · {fmtDateTime(news.published_at)}
              </p>
            </div>
            <div className="flex gap-2">
              <NewsAnalysisReview key={news.id} news={news} />
              <Button onClick={() => setEditing(true)}>Edit news thesis</Button>
            </div>
          </header>
          <Card
            title="News / source"
            action={
              news.url ? (
                <Button
                  variant="outline"
                  size="sm"
                  render={<a href={news.url} target="_blank" rel="noreferrer" />}
                >
                  Open source
                </Button>
              ) : undefined
            }
          >
            <div className="mb-4 flex flex-wrap gap-2">
              {news.category && <Pill tone="accent">{news.category}</Pill>}
              {news.tags.map((tag) => (
                <Pill key={tag}>#{tag}</Pill>
              ))}
            </div>
            <dl className="grid gap-4 sm:grid-cols-2">
              {(
                [
                  ["Original text", news.original_text],
                  ["Summary", news.summary],
                  ["Notes / thesis", news.notes],
                ] as const
              ).map(([label, value]) => (
                <div key={label}>
                  <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
                  <dd className="mt-1 whitespace-pre-wrap text-sm">{value || "Not recorded"}</dd>
                </div>
              ))}
            </dl>
            <p className="mt-4 text-xs text-muted-foreground">
              Created {fmtDateTime(news.created_at)} · Updated {fmtDateTime(news.updated_at)}
            </p>
          </Card>
          <section aria-label="Affected assets and predictions" className="space-y-4">
            <div>
              <h2 className="font-semibold">Affected assets & predictions</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                User and AI judgments are separate records. AI predictions appear when available;
                validation results remain pending.
              </p>
            </div>
            {!news.assets.length && (
              <Card>
                <p className="text-sm text-muted-foreground">
                  No affected assets yet. Edit the news thesis to add one.
                </p>
              </Card>
            )}
            {news.assets.map((asset) => (
              <Card
                key={asset.id}
                title={
                  <h3 className="font-semibold">
                    {asset.symbol} {asset.display_name && `· ${asset.display_name}`}
                  </h3>
                }
                description={[
                  asset.asset_type.toUpperCase(),
                  asset.market,
                  asset.exchange,
                  `Asset source: ${asset.source === "ai" ? "AI" : "User"}`,
                ]
                  .filter(Boolean)
                  .join(" · ")}
              >
                <p className="text-sm text-muted-foreground">
                  Relation: {asset.relation || "Not recorded"}
                </p>
                <NewsPredictions
                  detail
                  news={{
                    ...news,
                    assets: [asset],
                    predictions: news.predictions?.filter((p) => p.news_asset_id === asset.id),
                  }}
                  {...predictionActions}
                />
              </Card>
            ))}
          </section>
          {editing && (
            <NewsFormDialog
              item={news}
              onClose={() => setEditing(false)}
              onSave={async (value) => {
                await update.mutateAsync({
                  id: news.id,
                  body: value.body,
                  previousAssetIds: news.assets.map((a) => a.id),
                  assets: value.assets,
                });
              }}
            />
          )}
        </>
      )}
    </Page>
  );
}
