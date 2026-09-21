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
      setError(err instanceof Error ? err.message : "Analysis failed.");
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
      setError("Select at least one horizon for every accepted prediction.");
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
      setError(err instanceof Error ? err.message : "Could not save review.");
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
        Analyze with AI
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
                <DialogTitle>Review AI suggestions</DialogTitle>
                <DialogDescription>
                  Suggestions may be wrong. Select only what you want to save. Edited AI suggestions
                  retain AI identity; manual additions are User records.
                </DialogDescription>
              </div>
            </DialogHeader>
            <DialogBody>
              {busy && <p role="status">Analyzing news… No data has been saved.</p>}
              {error && (
                <p role="alert" className="mb-3 text-sm text-destructive">
                  {error}
                </p>
              )}
              {!busy && !analysis && <Button onClick={() => void start()}>Retry analysis</Button>}
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
                          Accept {field}
                        </label>
                        <p className="text-xs text-muted-foreground">
                          Current: {baseline[field] || "Not recorded"}
                        </p>
                        <FormTextarea
                          aria-label={`Suggested ${field}`}
                          value={analysis[field]}
                          onChange={(e) => setAnalysis({ ...analysis, [field]: e.target.value })}
                        />
                      </div>
                    ))}
                    {!assets.length && <p>No asset suggestions. You can add a manual asset.</p>}
                    {assets.map((a, i) => (
                      <section
                        key={i}
                        aria-label={`Suggestion ${i + 1}`}
                        className="space-y-3 rounded-lg border p-4"
                      >
                        <label className="flex items-center gap-2 text-sm font-semibold">
                          <input
                            type="checkbox"
                            checked={a.accepted}
                            onChange={(e) => updateAsset(i, { accepted: e.target.checked })}
                          />
                          Accept asset {i + 1} · {a.source === "ai" ? "AI" : "User"}
                        </label>
                        <div className="grid gap-3 sm:grid-cols-2">
                          <Field label="Type">
                            <NativeSelect
                              aria-label="Type"
                              value={a.asset_type}
                              onChange={(e) =>
                                updateAsset(i, {
                                  asset_type: e.target.value as DraftAsset["asset_type"],
                                })
                              }
                            >
                              {["stock", "etf", "index"].map((v) => (
                                <NativeSelectOption key={v} value={v}>
                                  {v.toUpperCase()}
                                </NativeSelectOption>
                              ))}
                            </NativeSelect>
                          </Field>
                          {(["symbol", "market", "exchange", "display_name"] as const).map(
                            (field) => (
                              <Field key={field} label={field.replace("_", " ")}>
                                <FormInput
                                  aria-label={field}
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
                          Include prediction
                        </label>
                        {a.include_prediction && (
                          <div className="space-y-3">
                            <div className="grid grid-cols-2 gap-3">
                              <Field label="Direction">
                                <NativeSelect
                                  aria-label="Direction"
                                  value={a.direction}
                                  onChange={(e) =>
                                    updateAsset(i, {
                                      direction: e.target.value as DraftAsset["direction"],
                                    })
                                  }
                                >
                                  {["bullish", "bearish", "neutral"].map((v) => (
                                    <NativeSelectOption key={v} value={v}>
                                      {v}
                                    </NativeSelectOption>
                                  ))}
                                </NativeSelect>
                              </Field>
                              <Field label="Confidence (%)">
                                <FormInput
                                  aria-label="Confidence (%)"
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
                              <Field key={field} label={field}>
                                <FormTextarea
                                  aria-label={field}
                                  required={field === "reasoning" && a.accepted}
                                  value={a[field]}
                                  onChange={(e) => updateAsset(i, { [field]: e.target.value })}
                                />
                              </Field>
                            ))}
                            <fieldset>
                              <legend className="mb-2 text-sm">Trading-day horizons</legend>
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
                                    {h}D
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
                          Reject suggestion
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
                      Add manual asset
                    </Button>
                  </fieldset>
                </form>
              )}
            </DialogBody>
            <DialogFooter>
              <Button variant="outline" disabled={busy || saving} onClick={() => setOpen(false)}>
                Cancel
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
                Save selected suggestions
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}
