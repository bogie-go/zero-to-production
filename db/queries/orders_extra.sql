-- name: ListOrdersForUser :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2;

-- name: GetOrderForUser :one
SELECT * FROM orders WHERE id = $1 AND user_id = $2;
