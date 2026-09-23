import { fmtPct } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { useLingui } from "@lingui/react/macro";
import { useState, type FormEvent } from "react";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/Dialog";
import { Field } from "@/components/Field";
import { FormInput, FormTextarea } from "@/components/FormInput";
import { fmtDateTime } from "@/lib/format";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Pill } from "@/components/Pill";
import { Button } from "@/components/ui/button";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import type { News, Prediction, PredictionBody } from "@/lib/api/news";

export interface PredictionActions {
  onSavePrediction: (
    newsId: string,
    body: PredictionBody,
    prediction?: Prediction,
  ) => Promise<void>;
  onDeletePrediction: (newsId: string, prediction: Prediction) => Promise<void>;
}

export function PredictionForm({
  news,
  prediction,
  onClose,
  onSave,
}: {
  news: News;
  prediction?: Prediction;
  onClose: () => void;
  onSave: PredictionActions["onSavePrediction"];
}) {
  const { t } = useLingui();
  const labels = {
    reasoning: t({ id: "news.reasoning", message: "Reasoning" }),
    catalysts: t({ id: "news.catalysts", message: "Catalysts" }),
    risks: t({ id: "news.risks", message: "Risks" }),
    invalidation: t({ id: "news.invalidation", message: "Invalidation" }),
    bullish: t({ id: "news.bullish", message: "Bullish" }),
    bearish: t({ id: "news.bearish", message: "Bearish" }),
    neutral: t({ id: "news.neutral", message: "Neutral" }),
    stock: t({ id: "news.stock", message: "Stock" }),
    etf: "ETF",
    index: t({ id: "news.index", message: "Index" }),
  };
  const [body, setBody] = useState<PredictionBody>(() =>
    prediction
      ? { ...prediction }
      : {
          news_asset_id: news.assets[0]?.id ?? "",
          direction: "neutral",
          confidence: null,
          reasoning: "",
          catalysts: "",
          risks: "",
          invalidation: "",
          horizons: [1],
        },
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!body.horizons.length) {
      setError(
        t({ id: "news.horizonRequired", message: "Select at least one trading-day horizon." }),
      );
      return;
    }
    if (
      body.confidence !== null &&
      (!Number.isInteger(body.confidence) || body.confidence < 0 || body.confidence > 100)
    ) {
      setError(
        t({
          id: "news.confidenceInvalid",
          message: "Confidence must be an integer from 0 to 100.",
        }),
      );
      return;
    }
    setSaving(true);
    setError("");
    try {
      await onSave(news.id, body, prediction);
      onClose();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : t({ id: "news.savePredictionFailed", message: "Could not save prediction." }),
      );
    } finally {
      setSaving(false);
    }
  };
  return (
    <Dialog open onOpenChange={(open) => !open && !saving && onClose()} modal="trap-focus">
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <div>
            <DialogTitle>
              {prediction
                ? t({ id: "news.editUserPrediction", message: "Edit User prediction" })
                : t({ id: "news.newUserPrediction", message: "New User prediction" })}
            </DialogTitle>
            <DialogDescription>
              {news.title} · {t({ id: "news.userJudgment", message: `User judgment` })}
            </DialogDescription>
          </div>
        </DialogHeader>
        <DialogBody>
          <form id="prediction-form" onSubmit={submit} className="flex flex-col gap-4">
            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}
            <Field label={t({ id: "news.affectedAsset", message: "Affected asset" })}>
              <NativeSelect
                aria-label={t({ id: "news.affectedAsset", message: "Affected asset" })}
                value={body.news_asset_id}
                disabled={!!prediction}
                required
                onChange={(e) => setBody({ ...body, news_asset_id: e.target.value })}
              >
                {news.assets.map((a) => (
                  <NativeSelectOption key={a.id} value={a.id}>
                    {a.symbol} · {labels[a.asset_type]} · {a.market || a.exchange || a.display_name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <div className="grid grid-cols-2 gap-4">
              <Field label={t({ id: "news.direction", message: "Direction" })}>
                <NativeSelect
                  aria-label={t({ id: "news.direction", message: "Direction" })}
                  value={body.direction}
                  onChange={(e) =>
                    setBody({ ...body, direction: e.target.value as PredictionBody["direction"] })
                  }
                >
                  <NativeSelectOption value="bullish">
                    {t({ id: "news.bullish", message: "Bullish" })}
                  </NativeSelectOption>
                  <NativeSelectOption value="bearish">
                    {t({ id: "news.bearish", message: "Bearish" })}
                  </NativeSelectOption>
                  <NativeSelectOption value="neutral">
                    {t({ id: "news.neutral", message: "Neutral" })}
                  </NativeSelectOption>
                </NativeSelect>
              </Field>
              <Field
                label={t({ id: "news.confidencePercent", message: "Confidence (%)" })}
                description={t({ id: "news.confidenceHint", message: "Optional, 0–100." })}
              >
                <FormInput
                  aria-label={t({ id: "news.confidencePercent", message: "Confidence (%)" })}
                  type="number"
                  min={0}
                  max={100}
                  step={1}
                  value={body.confidence ?? ""}
                  onChange={(e) =>
                    setBody({
                      ...body,
                      confidence: e.target.value === "" ? null : Number(e.target.value),
                    })
                  }
                />
              </Field>
            </div>
            <fieldset>
              <legend className="mb-2 text-sm font-medium">
                {t({ id: "news.horizons", message: "Trading-day horizons" })}
              </legend>
              <div className="flex flex-wrap gap-4">
                {[1, 3, 5, 10, 20].map((h) => (
                  <label key={h} className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={body.horizons.includes(h)}
                      onChange={(e) =>
                        setBody({
                          ...body,
                          horizons: e.target.checked
                            ? [...body.horizons, h].sort((a, b) => a - b)
                            : body.horizons.filter((v) => v !== h),
                        })
                      }
                    />
                    {t({ id: "news.days", message: `${h}D` })}
                  </label>
                ))}
              </div>
            </fieldset>
            {(["reasoning", "catalysts", "risks", "invalidation"] as const).map((key) => (
              <Field key={key} label={labels[key]}>
                <FormTextarea
                  aria-label={labels[key]}
                  value={body[key]}
                  onChange={(e) => setBody({ ...body, [key]: e.target.value })}
                />
              </Field>
            ))}
          </form>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" disabled={saving} onClick={onClose}>
            {t({ id: "news.cancel", message: "Cancel" })}
          </Button>
          <Button type="submit" form="prediction-form" loading={saving}>
            {t({ id: "news.savePrediction", message: "Save prediction" })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function NewsPredictions({
  news,
  onSavePrediction,
  onDeletePrediction,
  detail = false,
}: { news: News; detail?: boolean } & PredictionActions) {
  const { t } = useLingui();
  const labels = {
    reasoning: t({ id: "news.reasoning", message: "Reasoning" }),
    catalysts: t({ id: "news.catalysts", message: "Catalysts" }),
    risks: t({ id: "news.risks", message: "Risks" }),
    invalidation: t({ id: "news.invalidation", message: "Invalidation" }),
    bullish: t({ id: "news.bullish", message: "Bullish" }),
    bearish: t({ id: "news.bearish", message: "Bearish" }),
    neutral: t({ id: "news.neutral", message: "Neutral" }),
    stock: t({ id: "news.stock", message: "Stock" }),
    etf: "ETF",
    index: t({ id: "news.index", message: "Index" }),
  };
  const [editing, setEditing] = useState<Prediction | null | undefined>();
  const [deleting, setDeleting] = useState<Prediction>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <section
      aria-label={t({ id: "news.predictionsFor", message: `Predictions for ${news.title}` })}
      className="mt-4 space-y-3 border-t pt-3"
    >
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold">
          {t({ id: "news.predictions", message: "Predictions" })}
        </h3>
        <Button
          size="sm"
          variant="outline"
          disabled={!news.assets.length}
          onClick={() => setEditing(null)}
        >
          {t({ id: "news.addPrediction", message: "Add prediction" })}
        </Button>
      </div>
      {!news.assets.length && (
        <p className="text-xs text-muted-foreground">
          {t({
            id: "news.predictionAssetHint",
            message: "Add an affected asset before making a prediction.",
          })}
        </p>
      )}
      {!(news.predictions ?? []).length && (
        <p className="text-xs text-muted-foreground">
          {t({ id: "news.noPredictions", message: "No predictions yet." })}
        </p>
      )}
      {(news.predictions ?? []).map((p) => (
        <article
          key={p.id}
          aria-label={t({
            id: "news.predictionAria",
            message: `${p.source === "user" ? t({ id: "news.user", message: "User" }) : "AI"} prediction ${p.id}`,
          })}
          className="rounded-md bg-muted/40 p-3 text-sm"
        >
          <div className="flex flex-wrap items-center gap-2">
            <Pill>{p.source === "user" ? t({ id: "news.user", message: "User" }) : "AI"}</Pill>
            <strong>{news.assets.find((a) => a.id === p.news_asset_id)?.symbol}</strong>
            <span className="capitalize">{labels[p.direction]}</span>
            <span>
              {p.confidence === null
                ? t({ id: "news.confidenceUnset", message: "Confidence not set" })
                : fmtPct(p.confidence / 100, intlLocale())}
            </span>
            <span>
              {p.horizons.map((h) => `${t({ id: "news.days", message: `${h}D` })}`).join(" / ")}
            </span>
          </div>
          {p.reasoning && <p className="mt-2 whitespace-pre-wrap">{p.reasoning}</p>}
          {detail ? (
            <div className="mt-3 space-y-3">
              <p className="text-xs text-muted-foreground">
                {t({
                  id: "news.timestamps",
                  message: `Created ${fmtDateTime(p.created_at)} · Updated ${fmtDateTime(p.updated_at)}`,
                })}
              </p>
              <dl className="grid gap-3 sm:grid-cols-2">
                {(["catalysts", "risks", "invalidation"] as const).map((field) => (
                  <div key={field}>
                    <dt className="text-xs font-medium capitalize text-muted-foreground">
                      {labels[field]}
                    </dt>
                    <dd className="mt-1 whitespace-pre-wrap">
                      {p[field] || t({ id: "news.notRecorded", message: "Not recorded" })}
                    </dd>
                  </div>
                ))}
              </dl>
              <Table
                aria-label={t({
                  id: "news.validationAria",
                  message: `Validation horizons for ${p.source === "user" ? t({ id: "news.user", message: "User" }) : "AI"} prediction`,
                })}
              >
                <TableHeader>
                  <TableRow>
                    <TableHead>
                      {t({ id: "news.tradingHorizon", message: "Trading-day horizon" })}
                    </TableHead>
                    <TableHead>{t({ id: "news.status", message: "Status" })}</TableHead>
                    <TableHead>{t({ id: "news.result", message: "Result" })}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {p.horizons.map((h) => (
                    <TableRow key={h}>
                      <TableCell>{t({ id: "news.days", message: `${h}D` })}</TableCell>
                      <TableCell>
                        {t({ id: "news.pendingValidation", message: "Pending validation" })}
                      </TableCell>
                      <TableCell>
                        {t({ id: "news.notAvailable", message: "Not available" })}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <p className="mt-2 text-xs text-muted-foreground">
              {t({ id: "news.pendingValidation", message: "Pending validation" })}
            </p>
          )}
          {p.source === "user" && (
            <div className="mt-2 flex gap-2">
              <Button size="sm" variant="ghost" onClick={() => setEditing(p)}>
                {t({ id: "news.editPrediction", message: "Edit prediction" })}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  setDeleting(p);
                  setError("");
                }}
              >
                {t({ id: "news.deletePrediction", message: "Delete prediction" })}
              </Button>
            </div>
          )}
        </article>
      ))}
      {editing !== undefined && (
        <PredictionForm
          news={news}
          prediction={editing ?? undefined}
          onClose={() => setEditing(undefined)}
          onSave={onSavePrediction}
        />
      )}
      {deleting && (
        <Dialog
          open
          onOpenChange={(open) => !open && !busy && setDeleting(undefined)}
          modal="trap-focus"
        >
          <DialogContent className="max-w-md">
            <DialogHeader>
              <div>
                <DialogTitle>
                  {t({ id: "news.deletePredictionTitle", message: "Delete User prediction?" })}
                </DialogTitle>
                <DialogDescription>
                  {t({
                    id: "news.deletePredictionWarning",
                    message: "This removes the prediction and its horizons. This cannot be undone.",
                  })}
                </DialogDescription>
              </div>
            </DialogHeader>
            <DialogBody>
              {error && (
                <p role="alert" className="text-destructive">
                  {error}
                </p>
              )}
            </DialogBody>
            <DialogFooter>
              <Button variant="outline" disabled={busy} onClick={() => setDeleting(undefined)}>
                {t({ id: "news.cancel", message: "Cancel" })}
              </Button>
              <Button
                variant="destructive"
                loading={busy}
                onClick={async () => {
                  setBusy(true);
                  try {
                    await onDeletePrediction(news.id, deleting);
                    setDeleting(undefined);
                  } catch (err) {
                    setError(
                      err instanceof Error
                        ? err.message
                        : t({
                            id: "news.deletePredictionFailed",
                            message: "Could not delete prediction.",
                          }),
                    );
                  } finally {
                    setBusy(false);
                  }
                }}
              >
                {t({ id: "news.delete", message: "Delete" })}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </section>
  );
}
