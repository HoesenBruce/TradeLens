import { useLingui } from "@lingui/react/macro";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Page } from "@/components/Page";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { tradesApi } from "@/lib/api/trades";
import { preferencesApi } from "@/lib/api/preferences";
import { useAccounts } from "@/lib/hooks/useAccounts";
import { tagsApi } from "@/lib/api/tags";
import type { TradeDetail } from "@/lib/api/types";
import { useFilters } from "@/lib/filters";
import { useMe } from "@/lib/hooks/useMe";
import { useTrades } from "@/lib/hooks/useTrades";
import { parseJournalNotes } from "@/lib/journalNotes";
import { backlogScope, reviewPatch, reviewQueue, reviewCutoffKey } from "@/lib/reviewInbox";
import { TRADE_GRADES, gradeFromInt, intFromGrade } from "@/lib/tradeGrades";

export function ReviewInboxView() {
  const { t } = useLingui();
  const accountIds = useFilters((s) => s.accountIds);
  const me = useMe();
  const trades = useTrades({ account_id: accountIds?.join(","), status: "closed" });
  const [days, setDays] = useState(14);
  const [backlog, setBacklog] = useState(false);
  const [confirmDismiss, setConfirmDismiss] = useState<string | null>(null);
  const accounts = useAccounts();
  const prefs = useQuery({
    queryKey: ["review-preferences", me.data?.id],
    queryFn: preferencesApi.get,
    enabled: !!me.data,
  });
  const qc = useQueryClient();
  const selected = (accounts.data ?? []).filter((a) =>
    accountIds?.length ? accountIds.includes(a.id) : a.account_type !== "backtest",
  );
  const visible = (trades.data ?? []).filter((trade) => {
    const cutoff = prefs.data?.prefs[reviewCutoffKey(trade.account_id)];
    return (
      typeof cutoff !== "string" ||
      !Number.isFinite(Date.parse(cutoff)) ||
      !trade.closed_at ||
      Date.parse(trade.closed_at) > Date.parse(cutoff)
    );
  });
  const now = Date.now();
  const recentQueue = reviewQueue(visible, days, false, now);
  const backlogQueue = reviewQueue(visible, days, true, now);
  const dismiss = useMutation({
    mutationFn: (restore: boolean) =>
      preferencesApi.patch(
        Object.fromEntries(
          selected.map((a) => [
            reviewCutoffKey(a.id),
            restore ? null : new Date(Date.now() - days * 86400000).toISOString(),
          ]),
        ),
      ),
    onSuccess: async () => {
      setConfirmDismiss(null);
      await qc.invalidateQueries({ queryKey: ["review-preferences"] });
    },
  });
  const scope = `${backlogScope(me.data?.id ?? "", accountIds)}:${days}`;
  return (
    <Page>
      <h1 className="text-xl font-semibold">
        {t({ id: "review.title", message: "Review inbox" })}
      </h1>
      <p className="text-muted-foreground">
        {t({
          id: "review.hint",
          message:
            "Closed trades are reviewed only after an execution grade is saved. Uses the selected accounts; global date and trade filters do not apply.",
        })}
      </p>
      <label className="flex items-center gap-3">
        {t({ id: "review.days", message: "Recent days" })}
        <Input
          className="w-24"
          type="number"
          min={1}
          max={90}
          value={days}
          onChange={(e) => {
            const n = Number(e.target.value);
            if (Number.isInteger(n) && n >= 1 && n <= 90) setDays(n);
          }}
        />
      </label>
      <p className="text-sm text-muted-foreground">
        {t({
          id: "review.dismissScope",
          message:
            "Dismiss older trades from review counts and future alerts for the selected accounts, across devices. Trades remain ungraded. Restore brings them back.",
        })}
      </p>
      <div className="flex gap-2">
        <Button
          variant="outline"
          disabled={!prefs.isSuccess || dismiss.isPending || !selected.length}
          onClick={() => setConfirmDismiss(scope)}
        >
          {t({ id: "review.dismiss", message: "Dismiss backlog" })}
        </Button>
        <Button
          variant="outline"
          disabled={!prefs.isSuccess || dismiss.isPending || !selected.length}
          onClick={() => dismiss.mutate(true)}
        >
          {t({ id: "review.restore", message: "Restore backlog" })}
        </Button>
      </div>
      {confirmDismiss === scope && (
        <div role="alert">
          <p>
            {t({
              id: "review.confirmDismiss",
              message:
                "Stop counting ungraded trades older than the recent window for these accounts?",
            })}
          </p>
          <Button disabled={dismiss.isPending} onClick={() => dismiss.mutate(false)}>
            {t({ id: "review.confirm", message: "Confirm dismissal" })}
          </Button>
          <Button
            variant="outline"
            disabled={dismiss.isPending}
            onClick={() => setConfirmDismiss(null)}
          >
            {t({ id: "review.cancelDismiss", message: "Cancel dismissal" })}
          </Button>
        </div>
      )}
      {dismiss.isError && (
        <p role="alert">
          {t({ id: "review.dismissError", message: "Could not update backlog. Please retry." })}
        </p>
      )}
      <div className="flex gap-2">
        <Button variant={backlog ? "outline" : "default"} onClick={() => setBacklog(false)}>
          {t({ id: "review.recent", message: "Recent" })} ({recentQueue.length})
        </Button>
        <Button variant={backlog ? "default" : "outline"} onClick={() => setBacklog(true)}>
          {t({ id: "review.backlog", message: "Backlog" })} ({backlogQueue.length})
        </Button>
        <Button
          variant="outline"
          onClick={() => {
            setDays(14);
            setBacklog(false);
          }}
        >
          {t({ id: "review.reset", message: "Reset" })}
        </Button>
      </div>
      {trades.isError || prefs.isError || me.isError ? (
        <div role="alert">
          {t({ id: "review.loadError", message: "Could not load trades." })}
          <Button
            onClick={() => void Promise.all([trades.refetch(), prefs.refetch(), me.refetch()])}
          >
            {t({ id: "review.retry", message: "Retry" })}
          </Button>
        </div>
      ) : trades.isLoading || prefs.isPending ? (
        <p>{t({ id: "review.loading", message: "Loading…" })}</p>
      ) : (
        <ReviewSession
          key={`${scope}:${days}:${backlog}:${prefs.data?.updated_at}`}
          trades={backlog ? backlogQueue : recentQueue}
        />
      )}
    </Page>
  );
}

function ReviewSession({ trades }: { trades: { id: string }[] }) {
  const { t } = useLingui();
  const [queue] = useState(trades);
  const [index, setIndex] = useState(0);
  const [busy, setBusy] = useState(false);
  const current = queue[index];
  const detail = useQuery({
    queryKey: ["trade", current?.id],
    queryFn: () => tradesApi.get(current!.id),
    enabled: !!current,
  });
  return (
    <>
      <div className="flex gap-2">
        <Button
          variant="outline"
          disabled={busy || index === 0}
          onClick={() => setIndex((i) => i - 1)}
        >
          {t({ id: "review.back", message: "Back" })}
        </Button>
        <Button
          variant="outline"
          disabled={busy || !current}
          onClick={() => setIndex((i) => i + 1)}
        >
          {t({ id: "review.skip", message: "Skip" })}
        </Button>
      </div>
      {!current ? (
        <p role="status">
          {t({
            id: "review.done",
            message:
              "Queue complete. Skipped trades remain unreviewed; reopen this page to review them.",
          })}
        </p>
      ) : detail.isError ? (
        <div role="alert">
          {t({ id: "review.loadError", message: "Could not load trades." })}
          <Button onClick={() => void detail.refetch()}>
            {t({ id: "review.retry", message: "Retry" })}
          </Button>
        </div>
      ) : detail.data ? (
        <ReviewEditor
          key={current.id}
          trade={detail.data}
          onBusy={setBusy}
          onSaved={() => setIndex((i) => i + 1)}
        />
      ) : (
        <p>{t({ id: "review.loading", message: "Loading…" })}</p>
      )}
    </>
  );
}

function ReviewEditor({
  trade,
  onBusy,
  onSaved,
}: {
  trade: TradeDetail;
  onBusy: (busy: boolean) => void;
  onSaved: () => void;
}) {
  const { t } = useLingui();
  const qc = useQueryClient();
  const journal = parseJournalNotes(trade.notes);
  const [grade, setGrade] = useState(gradeFromInt(trade.trade_quality));
  const [notes, setNotes] = useState(journal.reviewNotes);
  const [mistakes, setMistakes] = useState(
    trade.tags.filter((tag) => tag.kind === "mistake").map((tag) => tag.id),
  );
  const tags = useQuery({ queryKey: ["tags", "mistake"], queryFn: () => tagsApi.list("mistake") });
  const save = useMutation({
    mutationFn: async () => {
      // Refresh before merging so edits in another surface are preserved.
      const latest = await tradesApi.get(trade.id);
      return tradesApi.patch(trade.id, reviewPatch(latest, intFromGrade(grade)!, notes, mistakes));
    },
    onMutate: () => onBusy(true),
    onSettled: () => onBusy(false),
    onSuccess: async () => {
      await Promise.all(
        ["trades", "trade", "analytics", "alerts"].map((key) =>
          qc.invalidateQueries({ queryKey: [key] }),
        ),
      );
      onSaved();
    },
  });
  function reset() {
    setGrade(gradeFromInt(trade.trade_quality));
    setNotes(journal.reviewNotes);
    setMistakes(trade.tags.filter((tag) => tag.kind === "mistake").map((tag) => tag.id));
    save.reset();
  }
  return (
    <section className="flex flex-col gap-4 rounded-lg border bg-card p-4">
      <h2 className="text-lg font-semibold">
        {trade.symbol} · {trade.direction} · {trade.closed_at?.slice(0, 10)}
      </h2>
      <p className="whitespace-pre-wrap text-sm text-muted-foreground">{trade.notes}</p>
      <fieldset>
        <legend>{t({ id: "review.grade", message: "Execution grade" })}</legend>
        <div className="flex gap-2">
          {TRADE_GRADES.map((g) => (
            <Button
              key={g}
              disabled={save.isPending}
              aria-pressed={grade === g}
              variant={grade === g ? "default" : "outline"}
              onClick={() => setGrade(g)}
            >
              {g}
            </Button>
          ))}
        </div>
      </fieldset>
      <label>
        {t({ id: "review.notes", message: "Review notes" })}
        <Textarea
          disabled={save.isPending}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
        />
      </label>
      <fieldset disabled={save.isPending}>
        <legend>{t({ id: "review.mistakes", message: "Mistakes" })}</legend>
        {[
          ...(tags.data ?? []).filter((tag) => tag.kind === "mistake"),
          ...trade.tags.filter(
            (tag) => tag.kind === "mistake" && !tags.data?.some((item) => item.id === tag.id),
          ),
        ].map((tag) => (
          <label key={tag.id} className="mr-4 inline-flex gap-2">
            <input
              type="checkbox"
              checked={mistakes.includes(tag.id)}
              onChange={(e) =>
                setMistakes((ids) =>
                  e.target.checked ? [...ids, tag.id] : ids.filter((id) => id !== tag.id),
                )
              }
            />
            {tag.name}
          </label>
        ))}
      </fieldset>
      {tags.isError && (
        <p role="alert">
          {t({
            id: "review.tagsError",
            message: "Could not load mistakes. Existing selections are preserved.",
          })}
        </p>
      )}
      {save.isError && (
        <p role="alert">
          {t({
            id: "review.saveError",
            message: "Save failed. Your draft is retained; retry or cancel.",
          })}
        </p>
      )}
      <div className="flex gap-2">
        <Button disabled={!grade || save.isPending} onClick={() => save.mutate()}>
          {t({ id: "review.save", message: "Save and next" })}
        </Button>
        <Button variant="outline" disabled={save.isPending} onClick={reset}>
          {t({ id: "review.cancel", message: "Cancel edits" })}
        </Button>
      </div>
    </section>
  );
}
