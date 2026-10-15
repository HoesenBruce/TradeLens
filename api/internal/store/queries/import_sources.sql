-- name: PutImportSourceContext :exec
INSERT INTO import_source_contexts (id, user_id, account_id, payload) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO NOTHING;

-- name: GetImportSourceRow :one
SELECT * FROM import_source_rows WHERE id = ? AND user_id = ?;

-- name: PutImportSourceRow :exec
INSERT INTO import_source_rows (id, user_id, account_id, batch_id, context_id, row_number, payload) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: PutImportSourceFill :exec
INSERT INTO import_source_fills (source_id, execution_id, leg, snapshot) VALUES (?, ?, ?, ?);

-- name: SupersedeImportSources :exec
UPDATE import_source_rows SET active = 0 WHERE id IN (SELECT source_id FROM import_source_fills WHERE execution_id = ?);

-- name: ListImportSourceRows :many
SELECT r.*, c.payload AS context_payload FROM import_source_rows r
JOIN import_source_contexts c ON c.id = r.context_id
WHERE r.batch_id = ? AND r.user_id = ? ORDER BY r.row_number;

-- name: ListImportSourceFills :many
SELECT f.* FROM import_source_fills f JOIN import_source_rows r ON r.id = f.source_id
WHERE r.batch_id = ? AND r.user_id = ? ORDER BY r.row_number, f.leg;

-- name: ListImportSourceCounts :many
SELECT batch_id, count(*) AS retained_rows, count(CASE WHEN active = 1 THEN 1 END) AS active_rows
FROM import_source_rows WHERE user_id = ? GROUP BY batch_id;

-- name: UpdateImportSourceSnapshot :exec
UPDATE import_source_fills SET snapshot = ? WHERE source_id = ? AND execution_id = ?;

-- name: ReplaceImportedExecution :execrows
UPDATE executions SET external_id = ?, symbol = ?, instrument_type = ?, side = ?, quantity = ?, price = ?,
fees = ?, commission = ?, executed_at = ?, multiplier = ?, details = ?, dedup_hash = ?
WHERE id = ? AND user_id = ? AND account_id = ?;
