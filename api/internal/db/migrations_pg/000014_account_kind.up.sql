ALTER TABLE accounts ADD COLUMN account_kind TEXT NOT NULL DEFAULT 'brokerage';
ALTER TABLE accounts ADD COLUMN capabilities TEXT NOT NULL DEFAULT '["cash"]';
UPDATE accounts SET account_kind = CASE WHEN account_type IN ('prop', 'backtest', 'paper') THEN account_type ELSE 'brokerage' END,
  capabilities = CASE account_type WHEN 'margin' THEN '["margin"]' WHEN 'prop' THEN '[]' WHEN 'backtest' THEN '[]' WHEN 'paper' THEN '[]' ELSE '["cash"]' END;
