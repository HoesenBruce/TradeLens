import { ApiError } from "@/lib/api/client";
import { useLingui } from "@lingui/react/macro";
import { useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
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
import { Button } from "@/components/ui/button";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import {
  newsAnalysisApi,
  type News,
  type NewsAnalysis,
  type NewsAnalysisReview,
} from "@/lib/api/news";

type DraftAsset = NewsAnalysisReview["assets"][number] & { accepted: boolean };

export function NewsAnalysisReview({ news }: { news: News }) {
  const { t } = useLingui();
  const labels = {
    symbol: t({ id: "news.symbol", message: "Symbol" }),
    market: t({ id: "news.market", message: "Market" }),
    exchange: t({ id: "news.exchange", message: "Exchange" }),
    display_name: t({ id: "news.display_name", message: "Display name" }),
    reasoning: t({ id: "news.reasoning", message: "Reasoning" }),
    catalysts: t({ id: "news.catalysts", message: "Catalysts" }),
    risks: t({ id: "news.risks", message: "Risks" }),
    stock: t({ id: "news.stock", message: "Stock" }),
    etf: "ETF",
    index: t({ id: "news.index", message: "Index" }),
    bullish: t({ id: "news.bullish", message: "Bullish" }),
    bearish: t({ id: "news.bearish", message: "Bearish" }),
    neutral: t({ id: "news.neutral", message: "Neutral" }),
  };
  const client = useQueryClient();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [analysis, setAnalysis] = useState<NewsAnalysis>();
  const [baseline, setBaseline] = useState({ summary: "", category: "" });
  const [acceptSummary, setAcceptSummary] = useState(false);
  const [acceptCategory, setAcceptCategory] = useState(false);
  const [assets, setAssets] = useState<DraftAsset[]>([]);
  const start = async () => {
    setBusy(true);
    setError("");
    setAnalysis(undefined);
    setAssets([]);
    setAcceptSummary(false);
    setAcceptCategory(false);
    setBaseline({ summary: news.summary, category: news.category });
    try {
      const result = await newsAnalysisApi.analyze(news.id);
      setAnalysis(result);
      setAssets(
        result.assets.map((a) => ({
          ...a,
          source: "ai",
          include_prediction: true,
          accepted: false,
        })),
      );
    } catch (err) {
      setError(
        err instanceof ApiError && err.code === "ai_unavailable"
          ? t({
              id: "news.aiUnavailable",
              message: "Configure and enable AI Coach in Settings first.",
            })
          : t({ id: "news.analysisFailed", message: "Analysis failed." }),
      );
    } finally {
      setBusy(false);
    }
  };
  const updateAsset = (index: number, changes: Partial<DraftAsset>) =>
    setAssets((current) => current.map((a, i) => (i === index ? { ...a, ...changes } : a)));
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!analysis) return;
    const selected = assets.filter((a) => a.accepted);
    if (selected.some((a) => a.include_prediction && !a.horizons.length)) {
      setError(
        t({
          id: "news.reviewHorizonRequired",
          message: "Select at least one horizon for every accepted prediction.",
        }),
      );
      return;
    }
    setSaving(true);
    setError("");
    try {
      await newsAnalysisApi.accept(news.id, {
        summary: acceptSummary ? analysis.summary : undefined,
        category: acceptCategory ? analysis.category : undefined,
        expected_summary: baseline.summary,
        expected_category: baseline.category,
        assets: selected.map(({ accepted: _accepted, ...a }) => a),
      });
      await client.invalidateQueries({ queryKey: ["news"] });
      setOpen(false);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : t({ id: "news.reviewFailed", message: "Could not save review." }),
      );
    } finally {
      setSaving(false);
    }
  };
  return (
    <>
      <Button
        variant="outline"
        onClick={() => {
          setOpen(true);
          void start();
        }}
      >
        {t({ id: "news.analyze", message: "Analyze with AI" })}
      </Button>
      {open && (
        <Dialog
          open
          onOpenChange={(value) => !value && !busy && !saving && setOpen(false)}
          modal="trap-focus"
        >
          <DialogContent className="max-w-3xl">
            <DialogHeader>
              <div>
                <DialogTitle>
                  {t({ id: "news.review", message: "Review AI suggestions" })}
                </DialogTitle>
                <DialogDescription>
                  {t({
                    id: "news.reviewHint",
                    message:
                      "Suggestions may be wrong. Select only what you want to save. Edited AI suggestions retain AI identity; manual additions are User records.",
                  })}
                </DialogDescription>
              </div>
            </DialogHeader>
            <DialogBody>
              {busy && (
                <p role="status">
                  {t({ id: "news.analyzing", message: "Analyzing news… No data has been saved." })}
                </p>
              )}
              {error && (
                <p role="alert" className="mb-3 text-sm text-destructive">
                  {error}
                </p>
              )}
              {!busy && !analysis && (
                <Button onClick={() => void start()}>
                  {t({ id: "news.retryAnalysis", message: "Retry analysis" })}
                </Button>
              )}
              {analysis && (
                <form id="news-analysis-review" onSubmit={submit} className="space-y-5">
                  <fieldset disabled={saving} className="space-y-5">
                    {(["summary", "category"] as const).map((field) => (
                      <div key={field} className="space-y-2">
                        <label className="flex items-center gap-2 text-sm font-medium">
                          <input
                            type="checkbox"
                            checked={field === "summary" ? acceptSummary : acceptCategory}
                            onChange={(e) =>
                              field === "summary"
                                ? setAcceptSummary(e.target.checked)
                                : setAcceptCategory(e.target.checked)
                            }
                          />
                          {field === "summary"
                            ? t({ id: "news.acceptSummary", message: "Accept summary" })
                            : t({ id: "news.acceptCategory", message: "Accept category" })}
                        </label>
                        <p className="text-xs text-muted-foreground">
                          {t({
                            id: "news.current",
                            message: `Current: ${baseline[field] || t({ id: "news.notRecorded", message: "Not recorded" })}`,
                          })}
                        </p>
                        <FormTextarea
                          aria-label={
                            field === "summary"
                              ? t({ id: "news.suggestedSummary", message: "Suggested summary" })
                              : t({ id: "news.suggestedCategory", message: "Suggested category" })
                          }
                          value={analysis[field]}
                          onChange={(e) => setAnalysis({ ...analysis, [field]: e.target.value })}
                        />
                      </div>
                    ))}
                    {!assets.length && (
                      <p>
                        {t({
                          id: "news.noSuggestions",
                          message: "No asset suggestions. You can add a manual asset.",
                        })}
                      </p>
                    )}
                    {assets.map((a, i) => (
                      <section
                        key={i}
                        aria-label={t({
                          id: "news.suggestionNumber",
                          message: `Suggestion ${i + 1}`,
                        })}
                        className="space-y-3 rounded-lg border p-4"
                      >
                        <label className="flex items-center gap-2 text-sm font-semibold">
                          <input
                            type="checkbox"
                            checked={a.accepted}
                            onChange={(e) => updateAsset(i, { accepted: e.target.checked })}
                          />
                          {t({ id: "news.acceptAsset", message: `Accept asset ${i + 1}` })} ·{" "}
                          {a.source === "ai" ? "AI" : t({ id: "news.user", message: "User" })}
                        </label>
                        <div className="grid gap-3 sm:grid-cols-2">
                          <Field label={t({ id: "news.type", message: "Type" })}>
                            <NativeSelect
                              aria-label={t({ id: "news.type", message: "Type" })}
                              value={a.asset_type}
                              onChange={(e) =>
                                updateAsset(i, {
                                  asset_type: e.target.value as DraftAsset["asset_type"],
                                })
                              }
                            >
                              {["stock", "etf", "index"].map((v) => (
                                <NativeSelectOption key={v} value={v}>
                                  {labels[v as keyof typeof labels]}
                                </NativeSelectOption>
                              ))}
                            </NativeSelect>
                          </Field>
                          {(["symbol", "market", "exchange", "display_name"] as const).map(
                            (field) => (
                              <Field key={field} label={labels[field]}>
                                <FormInput
                                  aria-label={labels[field]}
                                  required={field === "symbol" && a.accepted}
                                  value={a[field]}
                                  onChange={(e) => updateAsset(i, { [field]: e.target.value })}
                                />
                              </Field>
                            ),
                          )}
                        </div>
                        <label className="flex items-center gap-2 text-sm">
                          <input
                            type="checkbox"
                            checked={a.include_prediction}
                            onChange={(e) =>
                              updateAsset(i, { include_prediction: e.target.checked })
                            }
                          />
                          {t({ id: "news.includePrediction", message: "Include prediction" })}
                        </label>
                        {a.include_prediction && (
                          <div className="space-y-3">
                            <div className="grid grid-cols-2 gap-3">
                              <Field label={t({ id: "news.direction", message: "Direction" })}>
                                <NativeSelect
                                  aria-label={t({ id: "news.direction", message: "Direction" })}
                                  value={a.direction}
                                  onChange={(e) =>
                                    updateAsset(i, {
                                      direction: e.target.value as DraftAsset["direction"],
                                    })
                                  }
                                >
                                  {["bullish", "bearish", "neutral"].map((v) => (
                                    <NativeSelectOption key={v} value={v}>
                                      {labels[v as keyof typeof labels]}
                                    </NativeSelectOption>
                                  ))}
                                </NativeSelect>
                              </Field>
                              <Field
                                label={t({
                                  id: "news.confidencePercent",
                                  message: "Confidence (%)",
                                })}
                              >
                                <FormInput
                                  aria-label={t({
                                    id: "news.confidencePercent",
                                    message: "Confidence (%)",
                                  })}
                                  type="number"
                                  min={0}
                                  max={100}
                                  step={1}
                                  required={a.accepted}
                                  value={a.confidence}
                                  onChange={(e) =>
                                    updateAsset(i, { confidence: Number(e.target.value) })
                                  }
                                />
                              </Field>
                            </div>
                            {(["reasoning", "catalysts", "risks"] as const).map((field) => (
                              <Field key={field} label={labels[field]}>
                                <FormTextarea
                                  aria-label={labels[field]}
                                  required={field === "reasoning" && a.accepted}
                                  value={a[field]}
                                  onChange={(e) => updateAsset(i, { [field]: e.target.value })}
                                />
                              </Field>
                            ))}
                            <fieldset>
                              <legend className="mb-2 text-sm">
                                {t({ id: "news.horizons", message: "Trading-day horizons" })}
                              </legend>
                              <div className="flex gap-4">
                                {[1, 3, 5, 10, 20].map((h) => (
                                  <label key={h} className="flex items-center gap-1 text-sm">
                                    <input
                                      type="checkbox"
                                      checked={a.horizons.includes(h)}
                                      onChange={(e) =>
                                        updateAsset(i, {
                                          horizons: e.target.checked
                                            ? [...a.horizons, h].sort((x, y) => x - y)
                                            : a.horizons.filter((v) => v !== h),
                                        })
                                      }
                                    />
                                    {t({ id: "news.days", message: `${h}D` })}
                                  </label>
                                ))}
                              </div>
                            </fieldset>
                          </div>
                        )}
                        <Button
                          type="button"
                          size="sm"
                          variant="ghost"
                          onClick={() =>
                            setAssets((current) => current.filter((_, index) => index !== i))
                          }
                        >
                          {t({ id: "news.rejectSuggestion", message: "Reject suggestion" })}
                        </Button>
                      </section>
                    ))}
                    <Button
                      type="button"
                      variant="outline"
                      disabled={assets.length >= 20}
                      onClick={() =>
                        setAssets([
                          ...assets,
                          {
                            asset_type: "stock",
                            symbol: "",
                            market: "",
                            exchange: "",
                            display_name: "",
                            direction: "neutral",
                            confidence: 50,
                            reasoning: "",
                            catalysts: "",
                            risks: "",
                            horizons: [1],
                            source: "user",
                            include_prediction: true,
                            accepted: true,
                          },
                        ])
                      }
                    >
                      {t({ id: "news.addManualAsset", message: "Add manual asset" })}
                    </Button>
                  </fieldset>
                </form>
              )}
            </DialogBody>
            <DialogFooter>
              <Button variant="outline" disabled={busy || saving} onClick={() => setOpen(false)}>
                {t({ id: "news.cancel", message: "Cancel" })}
              </Button>
              <Button
                type="submit"
                form="news-analysis-review"
                loading={saving}
                disabled={
                  busy ||
                  !analysis ||
                  (!acceptSummary && !acceptCategory && !assets.some((a) => a.accepted))
                }
              >
                {t({ id: "news.saveSuggestions", message: "Save selected suggestions" })}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}
