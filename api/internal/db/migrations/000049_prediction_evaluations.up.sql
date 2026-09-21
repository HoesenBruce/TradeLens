CREATE TABLE IF NOT EXISTS prediction_evaluations (
 id TEXT PRIMARY KEY,
 prediction_id TEXT NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
 prediction_revision TEXT NOT NULL,
 horizon INTEGER NOT NULL,
 fingerprint TEXT NOT NULL,
 previous_id TEXT NOT NULL DEFAULT '',
 result_json TEXT NOT NULL,
 attempted_at TEXT NOT NULL,
 UNIQUE(prediction_id, prediction_revision, horizon, fingerprint)
);
CREATE INDEX IF NOT EXISTS idx_evaluations_prediction ON prediction_evaluations(prediction_id, attempted_at);
CREATE TABLE IF NOT EXISTS prediction_revisions (
 prediction_id TEXT PRIMARY KEY REFERENCES predictions(id) ON DELETE CASCADE,
 revision INTEGER NOT NULL
);
CREATE TRIGGER IF NOT EXISTS advance_prediction_revision AFTER UPDATE ON predictions
BEGIN
 INSERT INTO prediction_revisions(prediction_id,revision) VALUES(NEW.id,2)
 ON CONFLICT(prediction_id) DO UPDATE SET revision=revision+1;
END;
