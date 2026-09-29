-- name: GetAnnualGoal :one
SELECT *
FROM annual_goals
WHERE user_id = $1 AND year = $2;

-- name: UpsertAnnualGoal :one
INSERT INTO annual_goals (user_id, year, amount, currency, updated_at)
VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
ON CONFLICT(user_id, year) DO UPDATE SET
    amount = excluded.amount,
    currency = excluded.currency,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteAnnualGoal :execrows
DELETE FROM annual_goals
WHERE user_id = $1 AND year = $2;
