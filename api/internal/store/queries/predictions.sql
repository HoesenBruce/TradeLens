-- name: CreatePrediction :one
INSERT INTO predictions (id, news_asset_id, source, direction, confidence, reasoning, catalysts, risks, invalidation)
SELECT sqlc.arg(id), sqlc.arg(news_asset_id), sqlc.arg(source), sqlc.arg(direction), sqlc.narg(confidence),
       sqlc.arg(reasoning), sqlc.arg(catalysts), sqlc.arg(risks), sqlc.arg(invalidation)
WHERE EXISTS (
  SELECT 1 FROM news_assets JOIN news ON news.id = news_assets.news_id
  WHERE news_assets.id = sqlc.arg(news_asset_id) AND news.user_id = sqlc.arg(user_id)
)
RETURNING *;

-- name: GetPrediction :one
SELECT predictions.* FROM predictions
WHERE predictions.id = sqlc.arg(id) AND EXISTS (SELECT 1 FROM news_assets JOIN news ON news.id = news_assets.news_id
  WHERE news_assets.id = predictions.news_asset_id AND news.user_id = sqlc.arg(user_id));

-- name: ListPredictions :many
SELECT predictions.* FROM predictions
JOIN news_assets ON news_assets.id = predictions.news_asset_id
JOIN news ON news.id = news_assets.news_id
WHERE news.id = sqlc.arg(news_id) AND news.user_id = sqlc.arg(user_id)
ORDER BY predictions.created_at, predictions.id;

-- name: UpdatePrediction :one
UPDATE predictions SET direction = sqlc.arg(direction), confidence = sqlc.narg(confidence),
  reasoning = sqlc.arg(reasoning), catalysts = sqlc.arg(catalysts), risks = sqlc.arg(risks),
  invalidation = sqlc.arg(invalidation), updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now')
WHERE predictions.id = sqlc.arg(id) AND predictions.source = sqlc.arg(source) AND EXISTS (SELECT 1 FROM news_assets JOIN news ON news.id = news_assets.news_id
  WHERE news_assets.id = predictions.news_asset_id AND news.user_id = sqlc.arg(user_id))
RETURNING *;

-- name: DeletePrediction :execrows
DELETE FROM predictions
WHERE predictions.id = sqlc.arg(id) AND predictions.source = sqlc.arg(source) AND EXISTS (SELECT 1 FROM news_assets JOIN news ON news.id = news_assets.news_id
  WHERE news_assets.id = predictions.news_asset_id AND news.user_id = sqlc.arg(user_id));

-- name: AddPredictionHorizon :exec
INSERT INTO prediction_horizons (prediction_id, trading_days)
SELECT sqlc.arg(prediction_id), sqlc.arg(trading_days)
WHERE EXISTS (SELECT 1 FROM predictions WHERE predictions.id = sqlc.arg(prediction_id) AND EXISTS (SELECT 1 FROM news_assets JOIN news ON news.id = news_assets.news_id
  WHERE news_assets.id = predictions.news_asset_id AND news.user_id = sqlc.arg(user_id)));

-- name: ListPredictionHorizons :many
SELECT prediction_horizons.* FROM prediction_horizons
JOIN predictions ON predictions.id = prediction_horizons.prediction_id
WHERE prediction_id = sqlc.arg(prediction_id) AND EXISTS (SELECT 1 FROM news_assets JOIN news ON news.id = news_assets.news_id
  WHERE news_assets.id = predictions.news_asset_id AND news.user_id = sqlc.arg(user_id))
ORDER BY trading_days;

-- name: DeletePredictionHorizons :exec
DELETE FROM prediction_horizons WHERE prediction_id = sqlc.arg(prediction_id)
AND EXISTS (SELECT 1 FROM predictions WHERE predictions.id = sqlc.arg(prediction_id) AND EXISTS (SELECT 1 FROM news_assets JOIN news ON news.id = news_assets.news_id
  WHERE news_assets.id = predictions.news_asset_id AND news.user_id = sqlc.arg(user_id)));
