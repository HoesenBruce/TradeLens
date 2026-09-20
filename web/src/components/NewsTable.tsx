import { Fragment, useState } from "react";
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
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [expanded, setExpanded] = useState<string>();
  const pageCount = Math.max(1, Math.ceil(news.length / pageSize));
  const current = Math.min(page, pageCount);
  return (
    <>
      <Table aria-label="News theses">
        <TableHeader>
          <TableRow>
            {[
              "Published",
              "Title / source",
              "Affected assets",
              "Prediction / source",
              "Confidence",
              "Horizon",
              "Validation",
              "Actions",
            ].map((label) => (
              <TableHead key={label}>{label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {news.slice((current - 1) * pageSize, current * pageSize).map((n) => (
            <Fragment key={n.id}>
              <TableRow>
                <TableCell className="whitespace-nowrap text-xs">
                  {fmtDateTime(n.published_at)}
                </TableCell>
                <TableCell className="min-w-52 max-w-80 whitespace-normal">
                  <strong>{n.title}</strong>
                  <p className="mt-1 text-xs text-muted-foreground">{n.source}</p>
                </TableCell>
                <TableCell>
                  {n.assets.length
                    ? n.assets.map((a) => (
                        <div key={a.id}>
                          {a.symbol}{" "}
                          <span className="text-xs text-muted-foreground">
                            {a.market} {a.asset_type.toUpperCase()}
                          </span>
                        </div>
                      ))
                    : "—"}
                </TableCell>
                <TableCell>
                  {n.predictions?.length
                    ? n.predictions.map((p) => (
                        <div key={p.id} className="whitespace-nowrap capitalize">
                          {n.assets.find((a) => a.id === p.news_asset_id)?.symbol} · {p.direction} ·{" "}
                          {p.source === "user" ? "User" : "AI"}
                        </div>
                      ))
                    : "No prediction"}
                </TableCell>
                <TableCell>
                  {n.predictions?.length
                    ? n.predictions.map((p) => (
                        <div key={p.id}>
                          {p.confidence === null ? "Not set" : `${p.confidence}%`}
                        </div>
                      ))
                    : "—"}
                </TableCell>
                <TableCell>
                  {n.predictions?.length
                    ? n.predictions.map((p) => (
                        <div key={p.id} className="whitespace-nowrap">
                          {p.horizons.map((h) => `${h}D`).join(" / ")}
                        </div>
                      ))
                    : "—"}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">Pending validation</TableCell>
                <TableCell>
                  <div className="flex items-center gap-1">
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-expanded={expanded === n.id}
                      onClick={() => setExpanded(expanded === n.id ? undefined : n.id)}
                    >
                      Predictions
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Edit ${n.title}`}
                      onClick={() => onEdit(n)}
                    >
                      <Pencil aria-hidden />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Delete ${n.title}`}
                      onClick={() => onDelete(n)}
                    >
                      <Trash2 aria-hidden />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
              {expanded === n.id && predictionActions && (
                <TableRow>
                  <TableCell colSpan={8}>
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
        onPageChange={setPage}
        onPageSizeChange={(size) => {
          setPageSize(size);
          setPage(1);
        }}
        alwaysShow
      />
    </>
  );
}
