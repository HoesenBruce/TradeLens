import { AlertCircle, Newspaper, Plus, RefreshCw, X } from "lucide-react";
import { useState, type FormEvent } from "react";
import { Card } from "@/components/Card";
import { DateTimePicker } from "@/components/DateTimePicker";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/Dialog";
import { EmptyState } from "@/components/EmptyState";
import { Field } from "@/components/Field";
import { FormInput, FormTextarea } from "@/components/FormInput";
import { type PredictionActions } from "@/components/NewsPredictions";
import { Page } from "@/components/Page";
import { NewsTable } from "@/components/NewsTable";
import { filterNews, useNewsFilters } from "@/lib/newsFilters";
import { ListSkeleton } from "@/components/skeletons/list-skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import type { News, NewsAssetType, NewsBody } from "@/lib/api/news";
import { isoToWallClock, wallClockToIso } from "@/lib/displayPrefs";

import type { NewsAssetDraft } from "@/lib/hooks/useNews";

export interface NewsFormValue {
  body: NewsBody;
  assets: NewsAssetDraft[];
}

export interface NewsViewProps {
  predictionActions?: PredictionActions;
  news: News[];
  loading: boolean;
  error: boolean;
  onRetry: () => void;
  onSave: (value: NewsFormValue, item?: News) => Promise<void>;
  onDelete: (item: News) => Promise<void>;
}

interface AssetField extends NewsAssetDraft {
  key: number;
}

let nextAssetKey = 1;

function emptyAsset(): AssetField {
  return { key: nextAssetKey++, asset_type: "stock", symbol: "", source: "user" };
}

function initialAssets(item?: News): AssetField[] {
  return (
    item?.assets.map((asset) => ({
      key: nextAssetKey++,
      id: asset.id,
      asset_type: asset.asset_type,
      symbol: asset.symbol,
      market: asset.market,
      exchange: asset.exchange,
      display_name: asset.display_name,
      relation: asset.relation,
      source: asset.source,
    })) ?? []
  );
}

function parseTags(value: string): string[] {
  return [
    ...new Set(
      value
        .split(",")
        .map((tag) => tag.trim())
        .filter(Boolean),
    ),
  ];
}

export function NewsFormDialog({
  item,
  onClose,
  onSave,
}: {
  item?: News;
  onClose: () => void;
  onSave: (value: NewsFormValue, item?: News) => Promise<void>;
}) {
  const [title, setTitle] = useState(item?.title ?? "");
  const [source, setSource] = useState(item?.source ?? "");
  const [url, setURL] = useState(item?.url ?? "");
  const [publishedAt, setPublishedAt] = useState(isoToWallClock(item?.published_at ?? new Date()));
  const [originalText, setOriginalText] = useState(item?.original_text ?? "");
  const [notes, setNotes] = useState(item?.notes ?? "");
  const [summary, setSummary] = useState(item?.summary ?? "");
  const [category, setCategory] = useState(item?.category ?? "");
  const [tags, setTags] = useState(item?.tags.join(", ") ?? "");
  const [assets, setAssets] = useState<AssetField[]>(() => initialAssets(item));
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  const updateAsset = (key: number, patch: Partial<AssetField>) => {
    setAssets((current) =>
      current.map((asset) => (asset.key === key ? { ...asset, ...patch } : asset)),
    );
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setSaveError("");
    if (!title.trim() || !source.trim() || !publishedAt) {
      setSaveError("Title, source, and published time are required.");
      return;
    }
    if (assets.some((asset) => !asset.symbol.trim())) {
      setSaveError("Every affected asset needs a symbol.");
      return;
    }
    setSaving(true);
    try {
      await onSave(
        {
          body: {
            title: title.trim(),
            source: source.trim(),
            url: url.trim(),
            published_at: wallClockToIso(publishedAt),
            original_text: originalText.trim(),
            notes: notes.trim(),
            summary: summary.trim(),
            category: category.trim(),
            tags: parseTags(tags),
          },
          assets: assets.map(({ key: _key, ...asset }) => ({
            ...asset,
            symbol: asset.symbol.trim().toUpperCase(),
            market: asset.market?.trim(),
            exchange: asset.exchange?.trim(),
            display_name: asset.display_name?.trim(),
            relation: asset.relation?.trim(),
            source: "user",
          })),
        },
        item,
      );
      onClose();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Could not save news entry.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && !saving && onClose()} modal="trap-focus">
      <DialogContent className="max-w-[min(860px,94vw)]">
        <DialogHeader className="pr-12">
          <div>
            <DialogTitle>{item ? "Edit news thesis" : "New news thesis"}</DialogTitle>
            <DialogDescription className="mt-1">
              Record the source, your notes, and every affected stock, ETF, or index.
            </DialogDescription>
          </div>
        </DialogHeader>
        <DialogBody>
          <form id="news-form" className="flex flex-col gap-5" onSubmit={submit}>
            {saveError ? (
              <Alert variant="error">
                <AlertCircle aria-hidden />
                <AlertTitle>Could not save</AlertTitle>
                <AlertDescription>{saveError}</AlertDescription>
              </Alert>
            ) : null}

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Title" htmlFor="news-title">
                <FormInput
                  id="news-title"
                  aria-label="Title"
                  autoFocus
                  required
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  placeholder="What happened?"
                />
              </Field>
              <Field label="Source" htmlFor="news-source">
                <FormInput
                  id="news-source"
                  aria-label="Source"
                  required
                  value={source}
                  onChange={(event) => setSource(event.target.value)}
                  placeholder="Company filing, Reuters, manual…"
                />
              </Field>
              <Field label="Published time">
                <DateTimePicker
                  aria-label="Published time"
                  value={publishedAt}
                  onChange={setPublishedAt}
                />
              </Field>
              <Field label="URL" htmlFor="news-url" description="Optional HTTP(S) source link.">
                <FormInput
                  id="news-url"
                  aria-label="URL"
                  type="url"
                  value={url}
                  onChange={(event) => setURL(event.target.value)}
                  placeholder="https://…"
                />
              </Field>
              <Field label="Category" htmlFor="news-category">
                <FormInput
                  id="news-category"
                  aria-label="Category"
                  value={category}
                  onChange={(event) => setCategory(event.target.value)}
                  placeholder="Earnings, policy, launch…"
                />
              </Field>
              <Field label="Tags" htmlFor="news-tags" description="Comma-separated.">
                <FormInput
                  id="news-tags"
                  aria-label="Tags"
                  value={tags}
                  onChange={(event) => setTags(event.target.value)}
                  placeholder="Japan, ETF"
                />
              </Field>
            </div>

            <Field label="Original text" htmlFor="news-original">
              <FormTextarea
                id="news-original"
                aria-label="Original text"
                value={originalText}
                onChange={(event) => setOriginalText(event.target.value)}
                placeholder="Paste the relevant excerpt."
              />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Summary" htmlFor="news-summary">
                <FormTextarea
                  id="news-summary"
                  aria-label="Summary"
                  value={summary}
                  onChange={(event) => setSummary(event.target.value)}
                  placeholder="What changed?"
                />
              </Field>
              <Field label="Notes / thesis" htmlFor="news-notes">
                <FormTextarea
                  id="news-notes"
                  aria-label="Notes / thesis"
                  value={notes}
                  onChange={(event) => setNotes(event.target.value)}
                  placeholder="Why could this matter?"
                />
              </Field>
            </div>

            <section className="flex flex-col gap-3 rounded-lg bg-muted/45 p-3">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold">Affected assets</h3>
                  <p className="text-xs text-muted-foreground">Stocks, ETFs, and indices.</p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setAssets((a) => [...a, emptyAsset()])}
                >
                  <Plus aria-hidden /> Add asset
                </Button>
              </div>
              {assets.length === 0 ? (
                <p className="rounded-md bg-background px-3 py-4 text-center text-xs text-muted-foreground">
                  No affected assets yet.
                </p>
              ) : (
                <div className="flex flex-col gap-2">
                  {assets.map((asset, index) => (
                    <fieldset
                      key={asset.key}
                      className="grid gap-2 rounded-md bg-background p-3 md:grid-cols-2 xl:grid-cols-[7rem_8rem_7rem_7rem_1fr_1fr_auto]"
                    >
                      <legend className="sr-only">Affected asset {index + 1}</legend>
                      <Field label="Type">
                        <NativeSelect
                          aria-label={`Asset ${index + 1} type`}
                          value={asset.asset_type}
                          onChange={(event) =>
                            updateAsset(asset.key, {
                              asset_type: event.target.value as NewsAssetType,
                            })
                          }
                          wrapperClassName="w-full"
                        >
                          <NativeSelectOption value="stock">Stock</NativeSelectOption>
                          <NativeSelectOption value="etf">ETF</NativeSelectOption>
                          <NativeSelectOption value="index">Index</NativeSelectOption>
                        </NativeSelect>
                      </Field>
                      <Field label="Symbol">
                        <FormInput
                          aria-label={`Asset ${index + 1} symbol`}
                          required
                          value={asset.symbol}
                          onChange={(event) =>
                            updateAsset(asset.key, { symbol: event.target.value.toUpperCase() })
                          }
                          placeholder="285A"
                          className="uppercase"
                        />
                      </Field>
                      <Field label="Market">
                        <FormInput
                          value={asset.market ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { market: event.target.value })
                          }
                          placeholder="JP"
                        />
                      </Field>
                      <Field label="Exchange">
                        <FormInput
                          value={asset.exchange ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { exchange: event.target.value })
                          }
                          placeholder="TSE"
                        />
                      </Field>
                      <Field label="Name">
                        <FormInput
                          aria-label={`Asset ${index + 1} name`}
                          value={asset.display_name ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { display_name: event.target.value })
                          }
                          placeholder="Optional"
                        />
                      </Field>
                      <Field label="Relation">
                        <FormInput
                          aria-label={`Asset ${index + 1} relation`}
                          value={asset.relation ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { relation: event.target.value })
                          }
                          placeholder="Supplier, peer…"
                        />
                      </Field>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        className="self-end text-muted-foreground hover:text-destructive"
                        aria-label={`Remove asset ${index + 1}`}
                        onClick={() =>
                          setAssets((current) => current.filter((row) => row.key !== asset.key))
                        }
                      >
                        <X aria-hidden />
                      </Button>
                    </fieldset>
                  ))}
                </div>
              )}
            </section>
          </form>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button type="submit" form="news-form" loading={saving}>
            {item ? "Save changes" : "Create entry"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function DeleteNewsDialog({
  item,
  onClose,
  onDelete,
}: {
  item: News;
  onClose: () => void;
  onDelete: (item: News) => Promise<void>;
}) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");

  return (
    <Dialog open onOpenChange={(open) => !open && !deleting && onClose()} modal="trap-focus">
      <DialogContent className="max-w-md">
        <DialogHeader className="pr-12">
          <div>
            <DialogTitle>Delete news thesis?</DialogTitle>
            <DialogDescription className="mt-1">
              This also removes its affected assets and predictions. This cannot be undone.
            </DialogDescription>
          </div>
        </DialogHeader>
        <DialogBody className="gap-3">
          <p className="font-medium">{item.title}</p>
          {error ? (
            <Alert variant="error">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={deleting}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            loading={deleting}
            onClick={async () => {
              setDeleting(true);
              setError("");
              try {
                await onDelete(item);
                onClose();
              } catch (err) {
                setError(err instanceof Error ? err.message : "Could not delete entry.");
              } finally {
                setDeleting(false);
              }
            }}
          >
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function NewsView({
  news,
  loading,
  error,
  onRetry,
  onSave,
  onDelete,
  predictionActions,
}: NewsViewProps) {
  const [editing, setEditing] = useState<News | null | undefined>(undefined);
  const [deleting, setDeleting] = useState<News>();
  const { filters, setFilters, reset } = useNewsFilters();
  const filtered = filterNews(news, filters);

  return (
    <Page>
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">News thesis</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manual catalysts and the assets they may affect.
          </p>
        </div>
        <Button onClick={() => setEditing(null)}>
          <Plus aria-hidden /> New entry
        </Button>
      </header>

      <section
        aria-label="News filters"
        className="flex flex-wrap items-end gap-3 rounded-lg bg-card p-3"
      >
        <Field label="Validation">
          <NativeSelect
            aria-label="Validation filter"
            value={filters.status}
            onChange={(e) => setFilters({ status: e.target.value as typeof filters.status })}
          >
            <NativeSelectOption value="all">All statuses</NativeSelectOption>
            <NativeSelectOption value="pending">Pending</NativeSelectOption>
            <NativeSelectOption value="validated">Validated</NativeSelectOption>
          </NativeSelect>
        </Field>
        <Field label="Direction">
          <NativeSelect
            aria-label="Direction filter"
            value={filters.direction}
            onChange={(e) => setFilters({ direction: e.target.value as typeof filters.direction })}
          >
            {["all", "bullish", "bearish", "neutral"].map((d) => (
              <NativeSelectOption key={d} value={d}>
                {d === "all" ? "All directions" : d[0].toUpperCase() + d.slice(1)}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field label="Symbol">
          <FormInput
            aria-label="Symbol filter"
            placeholder="285A"
            value={filters.symbol}
            onChange={(e) => setFilters({ symbol: e.target.value })}
          />
        </Field>
        <Field label="From">
          <FormInput
            aria-label="Published from"
            type="date"
            value={filters.from}
            max={filters.to || undefined}
            onChange={(e) => setFilters({ from: e.target.value })}
          />
        </Field>
        <Field label="To">
          <FormInput
            aria-label="Published to"
            type="date"
            value={filters.to}
            min={filters.from || undefined}
            onChange={(e) => setFilters({ to: e.target.value })}
          />
        </Field>
        <Button variant="outline" onClick={reset}>
          Reset filters
        </Button>
      </section>

      {loading ? (
        <Card>
          <ListSkeleton rows={4} />
        </Card>
      ) : error ? (
        <Card>
          <Alert variant="error">
            <AlertCircle aria-hidden />
            <AlertTitle>Could not load news</AlertTitle>
            <AlertDescription>Check the API connection and try again.</AlertDescription>
            <Button variant="outline" size="sm" onClick={onRetry}>
              <RefreshCw aria-hidden /> Try again
            </Button>
          </Alert>
        </Card>
      ) : news.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Newspaper aria-hidden />}
            title="No news theses yet"
            hint="Record a catalyst and connect the stocks, ETFs, and indices it may affect."
            actions={
              <Button onClick={() => setEditing(null)}>
                <Plus aria-hidden /> New entry
              </Button>
            }
          />
        </Card>
      ) : (
        <Card title={`${filtered.length} ${filtered.length === 1 ? "entry" : "entries"}`} flush>
          {filtered.length ? (
            <NewsTable
              key={JSON.stringify(filters)}
              news={filtered}
              onEdit={setEditing}
              onDelete={setDeleting}
              predictionActions={predictionActions}
            />
          ) : (
            <div className="p-6">
              <EmptyState
                icon={<Newspaper aria-hidden />}
                title="No matching theses"
                hint="Change or reset filters to see more entries."
                actions={
                  <Button variant="outline" onClick={reset}>
                    Reset filters
                  </Button>
                }
              />
            </div>
          )}
        </Card>
      )}

      {editing !== undefined ? (
        <NewsFormDialog
          item={editing ?? undefined}
          onClose={() => setEditing(undefined)}
          onSave={onSave}
        />
      ) : null}
      {deleting ? (
        <DeleteNewsDialog
          item={deleting}
          onClose={() => setDeleting(undefined)}
          onDelete={onDelete}
        />
      ) : null}
    </Page>
  );
}
