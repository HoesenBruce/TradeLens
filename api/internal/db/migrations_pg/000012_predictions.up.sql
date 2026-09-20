CREATE TABLE IF NOT EXISTS predictions (
    id TEXT PRIMARY KEY,
    news_asset_id TEXT NOT NULL REFERENCES news_assets(id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('user', 'ai')),
    direction TEXT NOT NULL CHECK (direction IN ('bullish', 'bearish', 'neutral')),
    confidence INTEGER CHECK (confidence BETWEEN 0 AND 100),
    reasoning TEXT NOT NULL DEFAULT '',
    catalysts TEXT NOT NULL DEFAULT '',
    risks TEXT NOT NULL DEFAULT '',
    invalidation TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_predictions_asset ON predictions(news_asset_id, created_at, id);

CREATE TABLE IF NOT EXISTS prediction_horizons (
    prediction_id TEXT NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
    trading_days INTEGER NOT NULL CHECK (trading_days IN (1, 3, 5, 10, 20)),
    PRIMARY KEY (prediction_id, trading_days)
);
