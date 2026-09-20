-- name: CreateNews :one
INSERT INTO news (
  id, user_id, title, source, url, published_at, original_text, notes, summary, category, tags
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetNews :one
SELECT * FROM news WHERE id = ? AND user_id = ?;

-- name: ListNews :many
SELECT * FROM news
WHERE user_id = ?
ORDER BY published_at DESC, id DESC;

-- name: UpdateNews :one
UPDATE news SET
  title = ?, source = ?, url = ?, published_at = ?, original_text = ?, notes = ?,
  summary = ?, category = ?, tags = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING *;

-- name: DeleteNews :execrows
DELETE FROM news WHERE id = ? AND user_id = ?;

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
WHERE news_assets.id = ? AND news.user_id = ?;

-- name: ListNewsAssets :many
SELECT news_assets.* FROM news_assets
JOIN news ON news.id = news_assets.news_id
WHERE news_assets.news_id = ? AND news.user_id = ?
ORDER BY news_assets.id;

-- name: UpdateNewsAsset :one
UPDATE news_assets SET
  asset_type = ?, symbol = ?, market = ?, exchange = ?, display_name = ?, relation = ?,
  source = ?, updated_at = CURRENT_TIMESTAMP
WHERE news_assets.id = ? AND EXISTS (
  SELECT 1 FROM news WHERE news.id = news_assets.news_id AND news.user_id = ?
)
RETURNING *;

-- name: DeleteNewsAsset :execrows
DELETE FROM news_assets
WHERE news_assets.id = ? AND EXISTS (
  SELECT 1 FROM news WHERE news.id = news_assets.news_id AND news.user_id = ?
);
