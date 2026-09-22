import { fmtPct } from "@/lib/format";
import { intlLocale } from "@/lib/locale";
import { Link } from "@tanstack/react-router";
import { Fragment, useState } from "react";
import { useLingui } from "@lingui/react/macro";
import { NewsExportActions } from "@/components/NewsExportActions";
import { Pencil, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Pagination } from "@/components/Pagination";
import { NewsPredictions, type PredictionActions } from "@/components/NewsPredictions";
import type { News } from "@/lib/api/news";
import { fmtDateTime } from "@/lib/format";

export function NewsTable({
  news,
  onEdit,
  onDelete,
  predictionActions,
}: {
  news: News[];
  onEdit: (n: News) => void;
  onDelete: (n: News) => void;
  predictionActions?: PredictionActions;
}) {
  const { t } = useLingui();
  const labels = {
    bullish: t({ id: "news.bullish", message: "Bullish" }),
    bearish: t({ id: "news.bearish", message: "Bearish" }),
    neutral: t({ id: "news.neutral", message: "Neutral" }),
    stock: t({ id: "news.stock", message: "Stock" }),
    etf: "ETF",
    index: t({ id: "news.index", message: "Index" }),
  };
  const [selected, setSelected] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [expanded, setExpanded] = useState<string>();
  const pageCount = Math.max(1, Math.ceil(news.length / pageSize));
  const current = Math.min(page, pageCount);
  const rows = news.slice((current - 1) * pageSize, current * pageSize);
  const selectedIds = selected.filter((id) => rows.some((n) => n.id === id));
  return (
    <>
      <div className="p-3">
        <NewsExportActions ids={news.map((n) => n.id)} selected={selectedIds} />
      </div>
      <Table aria-label={t({ id: "news.table", message: "News theses" })}>
        <TableHeader>
          <TableRow>
            <TableHead>{t({ id: "news.export.select", message: "Select" })}</TableHead>
            {[
              t({ id: "news.published", message: "Published" }),
              t({ id: "news.titleSource", message: "Title / source" }),
              t({ id: "news.assets", message: "Affected assets" }),
              t({ id: "news.predictionSource", message: "Prediction / source" }),
              t({ id: "news.confidence", message: "Confidence" }),
              t({ id: "news.horizon", message: "Horizon" }),
              t({ id: "news.validation", message: "Validation" }),
              t({ id: "news.actions", message: "Actions" }),
            ].map((label) => (
              <TableHead key={label}>{label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((n) => (
            <Fragment key={n.id}>
              <TableRow>
                <TableCell>
                  <input
                    type="checkbox"
                    aria-label={t({
                      id: "news.export.selectTitle",
                      message: `Select ${n.title}`,
                    })}
                    checked={selectedIds.includes(n.id)}
                    onChange={(e) =>
                      setSelected(
                        e.target.checked
                          ? [...selectedIds, n.id]
                          : selectedIds.filter((id) => id !== n.id),
                      )
                    }
                  />
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs">
                  {fmtDateTime(n.published_at)}
                </TableCell>
                <TableCell className="min-w-52 max-w-80 whitespace-normal">
                  <Link
                    to="/news/$id"
                    params={{ id: n.id }}
                    className="font-semibold hover:underline"
                  >
                    {n.title}
                  </Link>
                  <p className="mt-1 text-xs text-muted-foreground">{n.source}</p>
                </TableCell>
                <TableCell>
                  {n.assets.length
                    ? n.assets.map((a) => (
                        <div key={a.id}>
                          {a.symbol}{" "}
                          <span className="text-xs text-muted-foreground">
                            {a.market} {labels[a.asset_type]}
                          </span>
                        </div>
                      ))
                    : "—"}
                </TableCell>
                <TableCell>
                  {n.predictions?.length
                    ? n.predictions.map((p) => (
                        <div key={p.id} className="whitespace-nowrap capitalize">
                          {n.assets.find((a) => a.id === p.news_asset_id)?.symbol} ·{" "}
                          {labels[p.direction]} ·{" "}
                          {p.source === "user" ? t({ id: "news.user", message: "User" }) : "AI"}
                        </div>
                      ))
                    : t({ id: "news.noPrediction", message: "No prediction" })}
                </TableCell>
                <TableCell>
                  {n.predictions?.length
                    ? n.predictions.map((p) => (
                        <div key={p.id}>
                          {p.confidence === null
                            ? t({ id: "news.notSet", message: "Not set" })
                            : fmtPct(p.confidence / 100, intlLocale())}
                        </div>
                      ))
                    : "—"}
                </TableCell>
                <TableCell>
                  {n.predictions?.length
                    ? n.predictions.map((p) => (
                        <div key={p.id} className="whitespace-nowrap">
                          {p.horizons
                            .map((h) => `${t({ id: "news.days", message: `${h}D` })}`)
                            .join(" / ")}
                        </div>
                      ))
                    : "—"}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {t({ id: "news.pendingValidation", message: "Pending validation" })}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-1">
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-expanded={expanded === n.id}
                      onClick={() => setExpanded(expanded === n.id ? undefined : n.id)}
                    >
                      {t({ id: "news.predictions", message: "Predictions" })}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={t({ id: "news.editTitle", message: `Edit ${n.title}` })}
                      onClick={() => onEdit(n)}
                    >
                      <Pencil aria-hidden />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={t({ id: "news.deleteNamedEntry", message: `Delete ${n.title}` })}
                      onClick={() => onDelete(n)}
                    >
                      <Trash2 aria-hidden />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
              {expanded === n.id && predictionActions && (
                <TableRow>
                  <TableCell colSpan={9}>
                    <NewsPredictions news={n} {...predictionActions} />
                  </TableCell>
                </TableRow>
              )}
            </Fragment>
          ))}
        </TableBody>
      </Table>
      <Pagination
        page={current}
        pageCount={pageCount}
        total={news.length}
        pageSize={pageSize}
        onPageChange={(next) => {
          setPage(next);
          setSelected([]);
        }}
        onPageSizeChange={(size) => {
          setPageSize(size);
          setSelected([]);
          setPage(1);
        }}
        alwaysShow
      />
    </>
  );
}
