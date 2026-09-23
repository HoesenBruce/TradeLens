import { useLingui } from "@lingui/react/macro";
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
import { NewsExportActions } from "@/components/NewsExportActions";
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
  const { t } = useLingui();
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
      setSaveError(
        t({ id: "news.required", message: "Title, source, and published time are required." }),
      );
      return;
    }
    if (assets.some((asset) => !asset.symbol.trim())) {
      setSaveError(
        t({ id: "news.assetRequired", message: "Every affected asset needs a symbol." }),
      );
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
      setSaveError(
        err instanceof Error
          ? err.message
          : t({ id: "news.saveFailed", message: "Could not save news entry." }),
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && !saving && onClose()} modal="trap-focus">
      <DialogContent className="max-w-[min(860px,94vw)]">
        <DialogHeader className="pr-12">
          <div>
            <DialogTitle>
              {item
                ? t({ id: "news.edit", message: "Edit news thesis" })
                : t({ id: "news.new", message: "New news thesis" })}
            </DialogTitle>
            <DialogDescription className="mt-1">
              {t({
                id: "news.formHint",
                message: "Record the source, your notes, and every affected stock, ETF, or index.",
              })}
            </DialogDescription>
          </div>
        </DialogHeader>
        <DialogBody>
          <form id="news-form" className="flex flex-col gap-5" onSubmit={submit}>
            {saveError ? (
              <Alert variant="error">
                <AlertCircle aria-hidden />
                <AlertTitle>{t({ id: "news.couldNotSave", message: "Could not save" })}</AlertTitle>
                <AlertDescription>{saveError}</AlertDescription>
              </Alert>
            ) : null}

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t({ id: "news.title", message: "Title" })} htmlFor="news-title">
                <FormInput
                  id="news-title"
                  aria-label={t({ id: "news.title", message: "Title" })}
                  autoFocus
                  required
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  placeholder={t({ id: "news.whatHappened", message: "What happened?" })}
                />
              </Field>
              <Field label={t({ id: "news.source", message: "Source" })} htmlFor="news-source">
                <FormInput
                  id="news-source"
                  aria-label={t({ id: "news.source", message: "Source" })}
                  required
                  value={source}
                  onChange={(event) => setSource(event.target.value)}
                  placeholder={t({
                    id: "news.sourceExample",
                    message: "Company filing, Reuters, manual…",
                  })}
                />
              </Field>
              <Field label={t({ id: "news.publishedTime", message: "Published time" })}>
                <DateTimePicker
                  aria-label={t({ id: "news.publishedTime", message: "Published time" })}
                  value={publishedAt}
                  onChange={setPublishedAt}
                />
              </Field>
              <Field
                label="URL"
                htmlFor="news-url"
                description={t({ id: "news.urlHint", message: "Optional HTTP(S) source link." })}
              >
                <FormInput
                  id="news-url"
                  aria-label="URL"
                  type="url"
                  value={url}
                  onChange={(event) => setURL(event.target.value)}
                  placeholder="https://…"
                />
              </Field>
              <Field
                label={t({ id: "news.category", message: "Category" })}
                htmlFor="news-category"
              >
                <FormInput
                  id="news-category"
                  aria-label={t({ id: "news.category", message: "Category" })}
                  value={category}
                  onChange={(event) => setCategory(event.target.value)}
                  placeholder={t({
                    id: "news.categoryExample",
                    message: "Earnings, policy, launch…",
                  })}
                />
              </Field>
              <Field
                label={t({ id: "news.tags", message: "Tags" })}
                htmlFor="news-tags"
                description={t({ id: "news.tagsHint", message: "Comma-separated." })}
              >
                <FormInput
                  id="news-tags"
                  aria-label={t({ id: "news.tags", message: "Tags" })}
                  value={tags}
                  onChange={(event) => setTags(event.target.value)}
                  placeholder={t({ id: "news.tagsExample", message: "Japan, ETF" })}
                />
              </Field>
            </div>

            <Field
              label={t({ id: "news.originalText", message: "Original text" })}
              htmlFor="news-original"
            >
              <FormTextarea
                id="news-original"
                aria-label={t({ id: "news.originalText", message: "Original text" })}
                value={originalText}
                onChange={(event) => setOriginalText(event.target.value)}
                placeholder={t({ id: "news.excerptHint", message: "Paste the relevant excerpt." })}
              />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t({ id: "news.summary", message: "Summary" })} htmlFor="news-summary">
                <FormTextarea
                  id="news-summary"
                  aria-label={t({ id: "news.summary", message: "Summary" })}
                  value={summary}
                  onChange={(event) => setSummary(event.target.value)}
                  placeholder={t({ id: "news.summaryHint", message: "What changed?" })}
                />
              </Field>
              <Field
                label={t({ id: "news.notes", message: "Notes / thesis" })}
                htmlFor="news-notes"
              >
                <FormTextarea
                  id="news-notes"
                  aria-label={t({ id: "news.notes", message: "Notes / thesis" })}
                  value={notes}
                  onChange={(event) => setNotes(event.target.value)}
                  placeholder={t({ id: "news.notesHint", message: "Why could this matter?" })}
                />
              </Field>
            </div>

            <section className="flex flex-col gap-3 rounded-lg bg-muted/45 p-3">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold">
                    {t({ id: "news.assets", message: "Affected assets" })}
                  </h3>
                  <p className="text-xs text-muted-foreground">
                    {t({ id: "news.assetHint", message: "Stocks, ETFs, and indices." })}
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setAssets((a) => [...a, emptyAsset()])}
                >
                  <Plus aria-hidden />
                  {t({ id: "news.addAsset", message: "Add asset" })}
                </Button>
              </div>
              {assets.length === 0 ? (
                <p className="rounded-md bg-background px-3 py-4 text-center text-xs text-muted-foreground">
                  {t({ id: "news.noAssets", message: "No affected assets yet." })}
                </p>
              ) : (
                <div className="flex flex-col gap-2">
                  {assets.map((asset, index) => (
                    <fieldset
                      key={asset.key}
                      className="grid gap-2 rounded-md bg-background p-3 md:grid-cols-2 xl:grid-cols-[7rem_8rem_7rem_7rem_1fr_1fr_auto]"
                    >
                      <legend className="sr-only">
                        {t({ id: "news.assetNumber", message: `Affected asset ${index + 1}` })}
                      </legend>
                      <Field label={t({ id: "news.type", message: "Type" })}>
                        <NativeSelect
                          aria-label={t({
                            id: "news.assetTypeNumber",
                            message: `Asset ${index + 1} type`,
                          })}
                          value={asset.asset_type}
                          onChange={(event) =>
                            updateAsset(asset.key, {
                              asset_type: event.target.value as NewsAssetType,
                            })
                          }
                          wrapperClassName="w-full"
                        >
                          <NativeSelectOption value="stock">
                            {t({ id: "news.stock", message: "Stock" })}
                          </NativeSelectOption>
                          <NativeSelectOption value="etf">ETF</NativeSelectOption>
                          <NativeSelectOption value="index">
                            {t({ id: "news.index", message: "Index" })}
                          </NativeSelectOption>
                        </NativeSelect>
                      </Field>
                      <Field label={t({ id: "news.symbol", message: "Symbol" })}>
                        <FormInput
                          aria-label={t({
                            id: "news.assetSymbol",
                            message: `Asset ${index + 1} symbol`,
                          })}
                          required
                          value={asset.symbol}
                          onChange={(event) =>
                            updateAsset(asset.key, { symbol: event.target.value.toUpperCase() })
                          }
                          placeholder="285A"
                          className="uppercase"
                        />
                      </Field>
                      <Field label={t({ id: "news.market", message: "Market" })}>
                        <FormInput
                          value={asset.market ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { market: event.target.value })
                          }
                          placeholder="JP"
                        />
                      </Field>
                      <Field label={t({ id: "news.exchange", message: "Exchange" })}>
                        <FormInput
                          value={asset.exchange ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { exchange: event.target.value })
                          }
                          placeholder="TSE"
                        />
                      </Field>
                      <Field label={t({ id: "news.name", message: "Name" })}>
                        <FormInput
                          aria-label={t({
                            id: "news.assetName",
                            message: `Asset ${index + 1} name`,
                          })}
                          value={asset.display_name ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { display_name: event.target.value })
                          }
                          placeholder={t({ id: "news.optional", message: "Optional" })}
                        />
                      </Field>
                      <Field label={t({ id: "news.relation", message: "Relation" })}>
                        <FormInput
                          aria-label={t({
                            id: "news.assetRelation",
                            message: `Asset ${index + 1} relation`,
                          })}
                          value={asset.relation ?? ""}
                          onChange={(event) =>
                            updateAsset(asset.key, { relation: event.target.value })
                          }
                          placeholder={t({ id: "news.relationHint", message: "Supplier, peer…" })}
                        />
                      </Field>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        className="self-end text-muted-foreground hover:text-destructive"
                        aria-label={t({
                          id: "news.removeAsset",
                          message: `Remove asset ${index + 1}`,
                        })}
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
            {t({ id: "news.cancel", message: "Cancel" })}
          </Button>
          <Button type="submit" form="news-form" loading={saving}>
            {item
              ? t({ id: "news.saveChanges", message: "Save changes" })
              : t({ id: "news.createEntry", message: "Create entry" })}
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
  const { t } = useLingui();
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");

  return (
    <Dialog open onOpenChange={(open) => !open && !deleting && onClose()} modal="trap-focus">
      <DialogContent className="max-w-md">
        <DialogHeader className="pr-12">
          <div>
            <DialogTitle>
              {t({ id: "news.deleteTitle", message: "Delete news thesis?" })}
            </DialogTitle>
            <DialogDescription className="mt-1">
              {t({
                id: "news.deleteWarning",
                message:
                  "This also removes its affected assets and predictions. This cannot be undone.",
              })}
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
            {t({ id: "news.cancel", message: "Cancel" })}
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
                setError(
                  err instanceof Error
                    ? err.message
                    : t({ id: "news.deleteFailed", message: "Could not delete entry." }),
                );
              } finally {
                setDeleting(false);
              }
            }}
          >
            {t({ id: "news.delete", message: "Delete" })}
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
  const { t } = useLingui();
  const labels = {
    bullish: t({ id: "news.bullish", message: "Bullish" }),
    bearish: t({ id: "news.bearish", message: "Bearish" }),
    neutral: t({ id: "news.neutral", message: "Neutral" }),
  };
  const [editing, setEditing] = useState<News | null | undefined>(undefined);
  const [deleting, setDeleting] = useState<News>();
  const { filters, setFilters, reset } = useNewsFilters();
  const filtered = filterNews(news, filters);

  return (
    <Page>
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">
            {t({ id: "news.heading", message: "News thesis" })}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t({
              id: "news.subtitle",
              message: "Manual catalysts and the assets they may affect.",
            })}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" render={<a href="/news/performance" />}>
            {t({ id: "news.performance", message: "Performance report" })}
          </Button>
          <Button onClick={() => setEditing(null)}>
            <Plus aria-hidden />
            {t({ id: "news.newEntry", message: "New entry" })}
          </Button>
        </div>
      </header>

      <section
        aria-label={t({ id: "news.filters", message: "News filters" })}
        className="flex flex-wrap items-end gap-3 rounded-lg bg-card p-3"
      >
        <Field label={t({ id: "news.validation", message: "Validation" })}>
          <NativeSelect
            aria-label={t({ id: "news.validationFilter", message: "Validation filter" })}
            value={filters.status}
            onChange={(e) => setFilters({ status: e.target.value as typeof filters.status })}
          >
            <NativeSelectOption value="all">
              {t({ id: "news.allStatuses", message: "All statuses" })}
            </NativeSelectOption>
            <NativeSelectOption value="pending">
              {t({ id: "news.pending", message: "Pending" })}
            </NativeSelectOption>
            <NativeSelectOption value="validated">
              {t({ id: "news.validated", message: "Validated" })}
            </NativeSelectOption>
          </NativeSelect>
        </Field>
        <Field label={t({ id: "news.direction", message: "Direction" })}>
          <NativeSelect
            aria-label={t({ id: "news.directionFilter", message: "Direction filter" })}
            value={filters.direction}
            onChange={(e) => setFilters({ direction: e.target.value as typeof filters.direction })}
          >
            {["all", "bullish", "bearish", "neutral"].map((d) => (
              <NativeSelectOption key={d} value={d}>
                {d === "all"
                  ? t({ id: "news.allDirections", message: "All directions" })
                  : labels[d as keyof typeof labels]}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field label={t({ id: "news.symbol", message: "Symbol" })}>
          <FormInput
            aria-label={t({ id: "news.symbolFilter", message: "Symbol filter" })}
            placeholder="285A"
            value={filters.symbol}
            onChange={(e) => setFilters({ symbol: e.target.value })}
          />
        </Field>
        <Field label={t({ id: "news.from", message: "From" })}>
          <FormInput
            aria-label={t({ id: "news.publishedFrom", message: "Published from" })}
            type="date"
            value={filters.from}
            max={filters.to || undefined}
            onChange={(e) => setFilters({ from: e.target.value })}
          />
        </Field>
        <Field label={t({ id: "news.to", message: "To" })}>
          <FormInput
            aria-label={t({ id: "news.publishedTo", message: "Published to" })}
            type="date"
            value={filters.to}
            min={filters.from || undefined}
            onChange={(e) => setFilters({ to: e.target.value })}
          />
        </Field>
        <Button variant="outline" onClick={reset}>
          {t({ id: "news.resetFilters", message: "Reset filters" })}
        </Button>
      </section>

      {!loading && !error && !filtered.length && <NewsExportActions />}

      {loading ? (
        <Card>
          <ListSkeleton rows={4} />
        </Card>
      ) : error ? (
        <Card>
          <Alert variant="error">
            <AlertCircle aria-hidden />
            <AlertTitle>{t({ id: "news.loadFailed", message: "Could not load news" })}</AlertTitle>
            <AlertDescription>
              {t({ id: "news.connectionHint", message: "Check the API connection and try again." })}
            </AlertDescription>
            <Button variant="outline" size="sm" onClick={onRetry}>
              <RefreshCw aria-hidden />
              {t({ id: "news.retry", message: "Try again" })}
            </Button>
          </Alert>
        </Card>
      ) : news.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Newspaper aria-hidden />}
            title={t({ id: "news.empty", message: "No news theses yet" })}
            hint={t({
              id: "news.emptyHint",
              message: "Record a catalyst and connect the stocks, ETFs, and indices it may affect.",
            })}
            actions={
              <Button onClick={() => setEditing(null)}>
                <Plus aria-hidden />
                {t({ id: "news.newEntry", message: "New entry" })}
              </Button>
            }
          />
        </Card>
      ) : (
        <Card title={t({ id: "news.entryCount", message: `Entries: ${filtered.length}` })} flush>
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
                title={t({ id: "news.noMatches", message: "No matching theses" })}
                hint={t({
                  id: "news.noMatchesHint",
                  message: "Change or reset filters to see more entries.",
                })}
                actions={
                  <Button variant="outline" onClick={reset}>
                    {t({ id: "news.resetFilters", message: "Reset filters" })}
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
