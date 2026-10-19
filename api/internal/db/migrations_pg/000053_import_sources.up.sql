CREATE TABLE IF NOT EXISTS import_source_contexts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    payload TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS import_source_rows (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    batch_id TEXT NOT NULL REFERENCES import_batches(id) ON DELETE CASCADE,
    context_id TEXT NOT NULL REFERENCES import_source_contexts(id),
    row_number INTEGER NOT NULL,
    payload TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_import_source_batch ON import_source_rows(user_id, batch_id);
CREATE TABLE IF NOT EXISTS import_source_fills (
    source_id TEXT NOT NULL REFERENCES import_source_rows(id) ON DELETE CASCADE,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    leg INTEGER NOT NULL,
    snapshot TEXT NOT NULL,
    PRIMARY KEY (source_id, leg)
);
CREATE INDEX IF NOT EXISTS idx_import_source_execution ON import_source_fills(execution_id);
