-- name: PutImportSourceContext :exec
INSERT INTO import_source_contexts (id, user_id, account_id, payload) VALUES ($1, $2, $3, $4) ON CONFLICT(id) DO NOTHING;

-- name: GetImportSourceRow :one
SELECT * FROM import_source_rows WHERE id = $1 AND user_id = $2;

-- name: PutImportSourceRow :exec
INSERT INTO import_source_rows (id, user_id, account_id, batch_id, context_id, row_number, payload) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: PutImportSourceFill :exec
INSERT INTO import_source_fills (source_id, execution_id, leg, snapshot) VALUES ($1, $2, $3, $4);

-- name: SupersedeImportSources :exec
UPDATE import_source_rows SET active = 0 WHERE id IN (SELECT source_id FROM import_source_fills WHERE execution_id = $1);

-- name: ListImportSourceRows :many
SELECT r.*, c.payload AS context_payload FROM import_source_rows r
JOIN import_source_contexts c ON c.id = r.context_id
WHERE r.batch_id = $1 AND r.user_id = $2 ORDER BY r.row_number;

-- name: ListImportSourceFills :many
SELECT f.* FROM import_source_fills f JOIN import_source_rows r ON r.id = f.source_id
WHERE r.batch_id = $1 AND r.user_id = $2 ORDER BY r.row_number, f.leg;

-- name: ListImportSourceCounts :many
SELECT batch_id, count(*) AS retained_rows, count(CASE WHEN active = 1 THEN 1 END) AS active_rows
FROM import_source_rows WHERE user_id = $1 GROUP BY batch_id;

-- name: UpdateImportSourceSnapshot :exec
UPDATE import_source_fills SET snapshot = $1 WHERE source_id = $2 AND execution_id = $3;

-- name: ReplaceImportedExecution :execrows
UPDATE executions SET external_id = $1, symbol = $2, instrument_type = $3, side = $4, quantity = $5, price = $6,
fees = $7, commission = $8, executed_at = $9, multiplier = $10, details = $11, dedup_hash = $12
WHERE id = $13 AND user_id = $14 AND account_id = $15;
