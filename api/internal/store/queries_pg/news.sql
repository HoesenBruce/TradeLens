-- name: CreateNews :one
INSERT INTO news (
  id, user_id, title, source, url, published_at, original_text, notes, summary, category, tags
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetNews :one
SELECT * FROM news WHERE id = $1 AND user_id = $2;

-- name: ListNews :many
SELECT * FROM news
WHERE user_id = $1
ORDER BY published_at DESC, id DESC;

-- name: UpdateNews :one
UPDATE news SET
  title = $1, source = $2, url = $3, published_at = $4, original_text = $5, notes = $6,
  summary = $7, category = $8, tags = $9, updated_at = CURRENT_TIMESTAMP
WHERE id = $10 AND user_id = $11
RETURNING *;

-- name: DeleteNews :execrows
DELETE FROM news WHERE id = $1 AND user_id = $2;

-- name: CreateNewsAsset :one
INSERT INTO news_assets (
  id, news_id, asset_type, symbol, market, exchange, display_name, relation, source
)
SELECT
  sqlc.arg(id), sqlc.arg(news_id), sqlc.arg(asset_type), sqlc.arg(symbol),
  sqlc.arg(market), sqlc.arg(exchange), sqlc.arg(display_name), sqlc.arg(relation), sqlc.arg(source)
WHERE EXISTS (
  SELECT 1 FROM news
  WHERE news.id = sqlc.arg(news_id) AND news.user_id = sqlc.arg(user_id)
)
RETURNING *;

-- name: GetNewsAsset :one
SELECT news_assets.* FROM news_assets
JOIN news ON news.id = news_assets.news_id
WHERE news_assets.id = $1 AND news.user_id = $2;

-- name: ListNewsAssets :many
SELECT news_assets.* FROM news_assets
JOIN news ON news.id = news_assets.news_id
WHERE news_assets.news_id = $1 AND news.user_id = $2
ORDER BY news_assets.id;

-- name: UpdateNewsAsset :one
UPDATE news_assets SET
  asset_type = $1, symbol = $2, market = $3, exchange = $4, display_name = $5, relation = $6,
  source = $7, updated_at = CURRENT_TIMESTAMP
WHERE news_assets.id = $8 AND EXISTS (
  SELECT 1 FROM news WHERE news.id = news_assets.news_id AND news.user_id = $9
)
RETURNING *;

-- name: DeleteNewsAsset :execrows
DELETE FROM news_assets
WHERE news_assets.id = $1 AND EXISTS (
  SELECT 1 FROM news WHERE news.id = news_assets.news_id AND news.user_id = $2
);
