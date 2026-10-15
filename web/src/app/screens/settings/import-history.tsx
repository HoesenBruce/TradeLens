import { useLingui } from "@lingui/react";
import { useLingui as useLinguiMacro } from "@lingui/react/macro";
import { intlLocale } from "@/lib/locale";
import { t as tr } from "@lingui/core/macro";
import { useState } from "react";
import { RotateCw, Undo2 } from "lucide-react";
import { Badge } from "@/components/reui/badge";
import { EmptyState } from "@/components/EmptyState";
import { Modal } from "@/components/Modal";
import { useToastManager } from "@/components/Toast";
import { Button } from "@/components/ui/button";
import type { Account, ImportBatch, ReprocessResult } from "@/lib/api/types";
import { useDeleteImport, useImports, useReprocessImport } from "@/lib/hooks/useImports";
import { SettingsGroup, SettingsSection } from "./settings-ui";

/** Human name for an import batch's source key. */
function sourceLabel(source: string): string {
  if (source === "ibkr-flex-sync")
    return tr({ id: "accounts.flexSource", message: "IBKR Flex sync" });
  if (source === "csv" || source === "json" || source === "file")
    return tr({ id: "accounts.fileSource", message: "File import" });
  return source;
}

function ReprocessButton({ batch }: { batch: ImportBatch }) {
  const { t: tr } = useLinguiMacro();
  const [open, setOpen] = useState(false);
  const [preview, setPreview] = useState<ReprocessResult | null>(null);
  const reprocess = useReprocessImport();
  const toast = useToastManager();

  function changeOpen(next: boolean) {
    if (reprocess.isPending) return;
    setOpen(next);
    setPreview(null);
    reprocess.reset();
  }

  function process(apply: boolean) {
    reprocess.mutate(
      { id: batch.id, fingerprint: apply ? preview?.fingerprint : undefined },
      {
        onSuccess: (result) => {
          if (result.applied) {
            setOpen(false);
            setPreview(null);
            toast.add({
              title: tr({ id: "imports.reprocessed", message: "Import reprocessed" }),
              description: tr({
                id: "imports.associationsKept",
                message: "Existing journal and review links were preserved.",
              }),
            });
          } else {
            setPreview(result);
          }
        },
        onError: () => setPreview(null),
      },
    );
  }

  return (
    <>
      <Button type="button" variant="outline" size="sm" onClick={() => changeOpen(true)}>
        <RotateCw size={13} strokeWidth={1.5} aria-hidden />
        {tr({ id: "imports.reprocess", message: "Reprocess" })}
      </Button>
      <Modal
        open={open}
        onOpenChange={changeOpen}
        title={tr({ id: "imports.reprocessTitle", message: "Reprocess with current rules" })}
        className="max-w-[min(520px,94vw)]"
        footer={
          <>
            <Button
              type="button"
              variant="ghost"
              disabled={reprocess.isPending}
              onClick={() => changeOpen(false)}
            >
              {tr({ id: "accounts.cancel", message: "Cancel" })}
            </Button>
            <Button
              type="button"
              disabled={reprocess.isPending}
              onClick={() => process(preview !== null)}
            >
              {reprocess.isPending
                ? tr({ id: "imports.processing", message: "Processing…" })
                : preview
                  ? tr({ id: "imports.applyReprocess", message: "Apply reprocessing" })
                  : tr({ id: "imports.previewReprocess", message: "Preview changes" })}
            </Button>
          </>
        }
      >
        <div className="space-y-3 text-[13px] leading-relaxed text-muted-foreground">
          <p className="m-0 break-words font-medium text-foreground">{batch.filename}</p>
          <p className="m-0">
            {tr({
              id: "imports.reprocessDescription",
              message:
                "Reparse retained source rows and recalculate this account using current rules. Historical P&L may change. Edits or grouping conflicts stop processing without changing your data.",
            })}
          </p>
          {preview ? (
            <div
              role="status"
              className="rounded-md border border-border bg-muted p-3 text-foreground"
            >
              {tr({
                id: "imports.reprocessSummary",
                message: `${preview.retained_rows} source rows · ${preview.executions} executions · ${preview.changed} changed · ${preview.trades_changed} trades changed`,
              })}
              <p className="mb-0 mt-2 text-muted-foreground">
                {tr({
                  id: "imports.associationsKept",
                  message: "Existing journal and review links were preserved.",
                })}
              </p>
            </div>
          ) : null}
          {reprocess.isError ? (
            <p role="alert" className="m-0 text-destructive">
              {reprocess.error.message}
            </p>
          ) : null}
        </div>
      </Modal>
    </>
  );
}

function RollbackButton({ batch, accountName }: { batch: ImportBatch; accountName: string }) {
  const { t: tr } = useLinguiMacro();

  const [open, setOpen] = useState(false);
  const toast = useToastManager();
  const rollback = useDeleteImport();

  function handleRollback() {
    rollback.mutate(batch.id, {
      onSuccess: () => {
        setOpen(false);
        toast.add({
          title: tr({ id: "accounts.rolledBackToast", message: "Import rolled back" }),
          description: tr({
            id: "accounts.removedRows",
            message: `${batch.row_count} rows removed`,
          }),
        });
      },
      onError: (err) =>
        toast.add({
          title: tr({ id: "accounts.rollbackError", message: "Could not roll back" }),
          description:
            err instanceof Error
              ? err.message
              : tr({ id: "accounts.rollbackFailed", message: "Rollback failed" }),
        }),
    });
  }

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
        className="border-border bg-transparent text-destructive hover:bg-destructive/10 hover:text-destructive"
      >
        <Undo2 size={13} strokeWidth={1.5} />
        {tr({ id: "accounts.rollback", message: "Roll back" })}
      </Button>
      <Modal
        open={open}
        onOpenChange={setOpen}
        title={tr({ id: "accounts.rollbackQuestion", message: "Roll back this import?" })}
        className="max-w-[min(420px,94vw)]"
        footer={
          <>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {tr({ id: "accounts.cancel", message: "Cancel" })}
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleRollback}
              disabled={rollback.isPending}
            >
              {rollback.isPending
                ? tr({ id: "accounts.rollingBack", message: "Rolling back…" })
                : tr({ id: "accounts.rollback", message: "Roll back" })}
            </Button>
          </>
        }
      >
        <p className="m-0 text-[13px] leading-relaxed text-muted-foreground">
          {tr({
            id: "accounts.rollbackDetails",
            message: `The ${batch.row_count} executions imported into ${accountName} by this batch will be deleted and trades regrouped. A later sync can import them again.`,
          })}
        </p>
      </Modal>
    </>
  );
}

/**
 * Every batch of imported executions — scheduled syncs, manual syncs, and
 * file imports — with per-batch rollback. Lives at the bottom of the
 * Accounts tab: a connection is a property of an account, and this is the
 * one cross-account view the batches need.
 */
export function ImportHistorySection({
  accounts,
  accountId,
}: {
  accounts: Account[];
  /** Limit the list to one account (the account detail page). */
  accountId?: string;
}) {
  "use no memo";
  useLingui();
  const { t: tr } = useLinguiMacro();

  const imports = useImports();

  const accountName = (id: string) =>
    accounts.find((a) => a.id === id)?.name ??
    tr({ id: "accounts.deletedAccount", message: "Deleted account" });
  const history = (imports.data ?? [])
    .filter((b) => !accountId || b.account_id === accountId)
    .sort((a, b) => b.created_at.localeCompare(a.created_at));

  return (
    <SettingsSection
      id="import-history"
      title={tr({ id: "accounts.history", message: "Sync & import history" })}
      description={tr({
        id: "accounts.historyHint",
        message:
          "Execution import rollbacks delete that batch's fills. SBI enrichment cannot be rolled back separately.",
      })}
    >
      {imports.isError ? (
        <EmptyState
          title={tr({ id: "accounts.historyError", message: "Could not load history" })}
          hint={tr({ id: "accounts.checkServer", message: "Check the server and try again." })}
        />
      ) : history.length === 0 ? (
        <EmptyState
          title={tr({ id: "accounts.noImports", message: "Nothing imported yet" })}
          hint={tr({
            id: "accounts.noImportsHint",
            message: "Batches appear here after a sync or file import.",
          })}
        />
      ) : (
        <SettingsGroup>
          {history.map((batch) => (
            <div key={batch.id} className="flex flex-wrap items-center gap-3 px-4 py-2.5">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="truncate text-[13px] font-medium">
                    {batch.source === "sbi_margin_pnl"
                      ? tr({
                          id: "accounts.sbiMarginPnLSource",
                          message: "SBI margin realized P&L",
                        })
                      : sourceLabel(batch.source)}
                  </span>
                  {batch.status === "reversed" ? (
                    <Badge variant="secondary">
                      {tr({ id: "accounts.rolledBack", message: "Rolled back" })}
                    </Badge>
                  ) : null}
                </div>
                <p className="m-0 text-[12px] text-muted-foreground">
                  {accountName(batch.account_id)}
                  {batch.filename ? ` · ${batch.filename}` : ""} ·{" "}
                  {tr({ id: "accounts.rowCount", message: `${batch.row_count} rows` })} ·{" "}
                  {new Date(batch.created_at).toLocaleString(intlLocale())}
                </p>
              </div>
              {batch.reprocessable ? (
                <ReprocessButton batch={batch} />
              ) : batch.status === "committed" ? (
                <span className="max-w-64 text-[11px] text-muted-foreground">
                  {(batch.retained_rows ?? 0) > 0
                    ? tr({
                        id: "imports.newerSource",
                        message: "Newer source data exists; use its import batch.",
                      })
                    : tr({
                        id: "imports.noRetainedSource",
                        message:
                          "No retained source rows. Older, duplicate-only or unsupported imports cannot be reprocessed here.",
                      })}
                </span>
              ) : null}
              {batch.source === "sbi_margin_pnl" ? (
                <span className="text-[11px] text-muted-foreground">
                  {tr({
                    id: "accounts.sbiEnrichmentHistory",
                    message: "This enrichment cannot be rolled back separately",
                  })}
                </span>
              ) : batch.status !== "reversed" ? (
                <RollbackButton batch={batch} accountName={accountName(batch.account_id)} />
              ) : null}
            </div>
          ))}
        </SettingsGroup>
      )}
    </SettingsSection>
  );
}
