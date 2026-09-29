import { plural } from "@lingui/core/macro";
import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { useEffect, useState } from "react";
import {
  useDeleteFlexSync,
  useFlexSync,
  useRunFlexSync,
  useSaveFlexSync,
} from "@/lib/hooks/useFlexSync";
import { Field } from "./Field";
import { FormInput, PasswordInput } from "./FormInput";
import { Modal } from "./Modal";
import { useToastManager } from "./Toast";
import { Button } from "./ui/button";
import { Switch } from "./ui/switch";

export interface FlexSyncButtonProps {
  accountId: string;
  accountName: string;
  /** Trigger label — defaults to "IBKR sync"; the Connections list uses "Configure". */
  label?: string;
}

/**
 * IBKR Flex auto-sync settings for one account: a Flex Web Service token and
 * Flex Query ID (Trades → Executions section, CSV format). Once enabled, the
 * background job pulls new fills on a schedule; "Sync now" runs one pass
 * immediately.
 */
export function FlexSyncButton({
  accountId,
  accountName,
  label = "IBKR sync",
}: FlexSyncButtonProps) {
  const { t: localize } = useSecondaryLingui();

  const [open, setOpen] = useState(false);
  const toast = useToastManager();
  const settingsQ = useFlexSync(accountId, open);
  const save = useSaveFlexSync(accountId);
  const remove = useDeleteFlexSync(accountId);
  const run = useRunFlexSync(accountId);

  const [queryId, setQueryId] = useState("");
  const [token, setToken] = useState("");
  const [enabled, setEnabled] = useState(true);

  const s = settingsQ.data;
  useEffect(() => {
    if (!s) return;
    setQueryId(s.query_id ?? "");
    setEnabled(s.configured ? s.enabled : true);
    setToken("");
  }, [s]);

  function handleSave() {
    if (queryId.trim() === "") {
      toast.add({
        title: localize({ id: "broker.queryIdIsRequired", message: "Query ID is required" }),
      });
      return;
    }
    if (!s?.token_set && token.trim() === "") {
      toast.add({
        title: localize({ id: "broker.tokenIsRequired", message: "Token is required" }),
        description: localize({
          id: "broker.pasteYourFlexWebServiceToken",
          message: "Paste your Flex Web Service token.",
        }),
      });
      return;
    }
    save.mutate(
      {
        query_id: queryId.trim(),
        enabled,
        ...(token.trim() !== "" ? { token: token.trim() } : {}),
      },
      {
        onSuccess: () =>
          toast.add({
            title: localize({ id: "broker.flexSyncSaved", message: "Flex sync saved" }),
            description: accountName,
          }),
        onError: (err) =>
          toast.add({
            title: localize({
              id: "broker.couldNotSaveFlexSync",
              message: "Could not save flex sync",
            }),
            description:
              err instanceof Error
                ? err.message
                : localize({ id: "trades.saveFailed", message: "Save failed" }),
          }),
      },
    );
  }

  function handleRun() {
    run.mutate(undefined, {
      onSuccess: (res) =>
        toast.add({
          title: localize({ id: "accounts.syncComplete", message: "Sync complete" }),
          description: localize({
            id: "broker.syncCounts",
            message: `${{ inserted: plural(res.inserted, { one: "# new execution", other: "# new executions" }) }}, ${{ skipped: plural(res.skipped, { one: "# duplicate", other: "# duplicates" }) }}`,
          }),
        }),
      onError: (err) =>
        toast.add({
          title: localize({ id: "accounts.syncFailed", message: "Sync failed" }),
          description:
            err instanceof Error
              ? err.message
              : localize({ id: "accounts.syncFailed", message: "Sync failed" }),
        }),
    });
  }

  function handleDisconnect() {
    remove.mutate(undefined, {
      onSuccess: () => {
        toast.add({
          title: localize({ id: "broker.flexSyncRemoved", message: "Flex sync removed" }),
          description: accountName,
        });
        setQueryId("");
        setToken("");
      },
    });
  }

  return (
    <>
      <Button type="button" variant="outline" size="sm" onClick={() => setOpen(true)}>
        {label}
      </Button>
      <Modal
        open={open}
        onOpenChange={setOpen}
        title={localize({
          id: "broker.ibkrFlexSyncValue0",
          message: `IBKR Flex sync — ${{ value0: accountName }}`,
        })}
        className="max-w-[min(460px,94vw)]"
        footer={
          <>
            {s?.configured ? (
              <Button
                type="button"
                variant="ghost"
                className="mr-auto text-destructive"
                onClick={handleDisconnect}
                disabled={remove.isPending}
              >
                {localize({ id: "trades.remove", message: "Remove" })}
              </Button>
            ) : null}
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {localize({ id: "trades.cancel", message: "Cancel" })}
            </Button>
            <Button type="button" onClick={handleSave} disabled={save.isPending}>
              {save.isPending
                ? localize({ id: "market.saving", message: "Saving…" })
                : localize({ id: "trades.save", message: "Save" })}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-3">
          <p className="m-0 text-[12px] leading-relaxed text-muted-foreground">
            {localize({
              id: "broker.inIbkrClientPortalCreateAFlexQueryWithTheTradesExecutions",
              message:
                "In IBKR Client Portal, create a Flex Query with the Trades → Executions section in CSV format, and enable the Flex Web Service to get a token. New fills import automatically and duplicates are skipped.",
            })}
          </p>
          <Field label={localize({ id: "broker.flexQueryId", message: "Flex Query ID" })}>
            <FormInput
              value={queryId}
              onChange={(e) => setQueryId(e.target.value)}
              placeholder="123456"
              aria-label={localize({ id: "broker.flexQueryId", message: "Flex Query ID" })}
            />
          </Field>
          <Field
            label={localize({
              id: "broker.flexWebServiceToken",
              message: "Flex Web Service token",
            })}
            description={
              s?.token_set
                ? localize({
                    id: "broker.savedValue0LeaveBlankToKeepIt",
                    message: `Saved (${{ value0: s.token_hint ?? "hidden" }}) — leave blank to keep it.`,
                  })
                : undefined
            }
          >
            <PasswordInput
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder={
                s?.token_set
                  ? "••••••••"
                  : localize({ id: "broker.pasteToken", message: "Paste token" })
              }
              aria-label={localize({
                id: "broker.flexWebServiceToken",
                message: "Flex Web Service token",
              })}
            />
          </Field>
          <Field label={localize({ id: "broker.scheduledSync", message: "Scheduled sync" })}>
            <div className="flex items-center gap-2">
              <Switch
                checked={enabled}
                onCheckedChange={setEnabled}
                aria-label={localize({ id: "broker.scheduledSync", message: "Scheduled sync" })}
              />
              <span className="text-[12px] text-muted-foreground">
                {enabled
                  ? localize({
                      id: "broker.runsAutomaticallyInTheBackground",
                      message: "Runs automatically in the background",
                    })
                  : localize({ id: "broker.manualSyncOnly", message: "Manual sync only" })}
              </span>
            </div>
          </Field>

          {s?.configured ? (
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between gap-3">
                <p className="m-0 text-[12px] text-muted-foreground">
                  {s.last_synced_at
                    ? localize({
                        id: "broker.lastSyncValue0Value1",
                        message: `Last sync ${{ value0: new Date(s.last_synced_at).toLocaleString() }} — ${{ value1: s.last_status || "done" }}`,
                      })
                    : localize({ id: "accounts.neverSynced", message: "Never synced yet." })}
                </p>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleRun}
                  disabled={run.isPending}
                >
                  {run.isPending
                    ? localize({ id: "accounts.syncing", message: "Syncing…" })
                    : localize({ id: "accounts.syncNow", message: "Sync now" })}
                </Button>
              </div>
              {s.last_error ? (
                <p className="m-0 text-[12px] text-destructive">{s.last_error}</p>
              ) : null}
            </div>
          ) : null}
        </div>
      </Modal>
    </>
  );
}
