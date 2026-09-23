-- name: CreateAccount :one
INSERT INTO accounts (id, user_id, name, broker, account_type, base_currency, starting_balance, account_kind, capabilities)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: ListAccounts :many
SELECT * FROM accounts WHERE user_id = $1 ORDER BY created_at;

-- name: GetAccount :one
SELECT * FROM accounts WHERE id = $1 AND user_id = $2;

-- name: UpdateAccount :one
UPDATE accounts
SET name = $1, broker = $2, account_type = $3, base_currency = $4, starting_balance = $5, account_kind = $6, capabilities = $7
WHERE id = $8 AND user_id = $9
RETURNING *;

-- name: DeleteAccount :execrows
DELETE FROM accounts WHERE id = $1 AND user_id = $2;

-- name: GetAccountByIDAny :one
SELECT * FROM accounts WHERE id = $1;
