import { useRef, useState } from "react";
import { useLingui } from "@lingui/react/macro";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";
import { downloadNews } from "@/lib/api/news";

export function NewsExportActions({
  id,
  ids = [],
  selected = [],
}: {
  id?: string;
  ids?: string[];
  selected?: string[];
}) {
  const { t } = useLingui();
  const lock = useRef(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const run = async (target: string | string[]) => {
    if (lock.current) return;
    lock.current = true;
    setBusy(true);
    setMessage("");
    try {
      await downloadNews(target);
      setMessage(
        t({
          id: "news.export.started",
          message: "Download requested. Check your browser downloads.",
        }),
      );
    } catch (error) {
      setMessage(
        error instanceof ApiError && error.code === "empty_export"
          ? t({ id: "news.export.empty", message: "No theses to export." })
          : error instanceof ApiError && (error.status === 401 || error.status === 403)
            ? t({
                id: "news.export.denied",
                message: "Export unavailable. Sign in and check your access.",
              })
            : error instanceof ApiError && error.status === 404
              ? t({
                  id: "news.export.missing",
                  message: "A thesis is no longer available. Refresh the list and try again.",
                })
              : t({
                  id: "news.export.failed",
                  message: "Could not export. Check your connection and selection, then try again.",
                }),
      );
    } finally {
      lock.current = false;
      setBusy(false);
    }
  };
  const count = selected.length;
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2" aria-busy={busy}>
        {id ? (
          <Button variant="outline" disabled={busy} onClick={() => void run(id)}>
            {t({ id: "news.export.single", message: "Export Markdown" })}
          </Button>
        ) : (
          <>
            <Button variant="outline" disabled={busy || !ids.length} onClick={() => void run(ids)}>
              {t({ id: "news.export.filtered", message: "Export filtered" })}
            </Button>
            <Button variant="outline" disabled={busy || !count} onClick={() => void run(selected)}>
              {t({ id: "news.export.selected", message: `Export selected (${count})` })}
            </Button>
          </>
        )}
        {busy && (
          <span role="status">{t({ id: "news.export.loading", message: "Exporting…" })}</span>
        )}
      </div>
      {!id && (
        <p className="text-xs text-muted-foreground">
          {t({
            id: "news.export.selectionHint",
            message:
              "Selection is page-local; changing page, page size, or filters clears it. Filtered export includes all matching pages.",
          })}
        </p>
      )}
      {!id && !ids.length && (
        <p role="status">{t({ id: "news.export.empty", message: "No theses to export." })}</p>
      )}
      {message && (
        <p role="status" className="text-sm">
          {message}
        </p>
      )}
    </div>
  );
}
