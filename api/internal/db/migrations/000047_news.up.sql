CREATE TABLE IF NOT EXISTS news (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    source        TEXT NOT NULL,
    url           TEXT NOT NULL DEFAULT '',
    published_at  TIMESTAMP NOT NULL,
    original_text TEXT NOT NULL DEFAULT '',
    notes         TEXT NOT NULL DEFAULT '',
    summary       TEXT NOT NULL DEFAULT '',
    category      TEXT NOT NULL DEFAULT '',
    tags          TEXT NOT NULL DEFAULT '[]',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_news_user_published ON news(user_id, published_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS news_assets (
    id           TEXT PRIMARY KEY,
    news_id      TEXT NOT NULL REFERENCES news(id) ON DELETE CASCADE,
    asset_type   TEXT NOT NULL,
    symbol       TEXT NOT NULL,
    market       TEXT NOT NULL DEFAULT '',
    exchange     TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    relation     TEXT NOT NULL DEFAULT '',
    source       TEXT NOT NULL DEFAULT 'user',
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_news_assets_news ON news_assets(news_id, id);
