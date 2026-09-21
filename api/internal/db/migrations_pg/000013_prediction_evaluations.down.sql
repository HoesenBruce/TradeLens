DROP TRIGGER IF EXISTS advance_prediction_revision ON predictions;
DROP FUNCTION IF EXISTS advance_prediction_revision();
DROP TABLE IF EXISTS prediction_revisions;
DROP TABLE IF EXISTS prediction_evaluations;
