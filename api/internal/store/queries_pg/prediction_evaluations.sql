-- name: SavePredictionEvaluation :one
INSERT INTO prediction_evaluations (id,prediction_id,prediction_revision,horizon,fingerprint,previous_id,result_json,attempted_at)
SELECT sqlc.arg(id),sqlc.arg(prediction_id),sqlc.arg(prediction_revision),sqlc.arg(horizon),sqlc.arg(fingerprint),sqlc.arg(previous_id),sqlc.arg(result_json),sqlc.arg(attempted_at)
WHERE EXISTS (SELECT 1 FROM predictions JOIN news_assets ON news_assets.id=predictions.news_asset_id JOIN news ON news.id=news_assets.news_id WHERE predictions.id=sqlc.arg(prediction_id) AND news.user_id=sqlc.arg(user_id))
ON CONFLICT(prediction_id,prediction_revision,horizon,fingerprint) DO UPDATE SET attempted_at=excluded.attempted_at
RETURNING *;

-- name: ListPredictionEvaluations :many
SELECT prediction_evaluations.* FROM prediction_evaluations
JOIN predictions ON predictions.id=prediction_evaluations.prediction_id
JOIN news_assets ON news_assets.id=predictions.news_asset_id
JOIN news ON news.id=news_assets.news_id
WHERE prediction_id=sqlc.arg(prediction_id) AND news.user_id=sqlc.arg(user_id)
ORDER BY attempted_at DESC, prediction_evaluations.id;

-- name: GetPredictionRevision :one
SELECT CAST(COALESCE((SELECT revision FROM prediction_revisions WHERE prediction_id=predictions.id),1) AS INTEGER) AS revision
FROM predictions JOIN news_assets ON news_assets.id=predictions.news_asset_id JOIN news ON news.id=news_assets.news_id
WHERE predictions.id=sqlc.arg(prediction_id) AND news.user_id=sqlc.arg(user_id);
