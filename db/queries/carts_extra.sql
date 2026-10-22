-- name: GetCartByToken :one
SELECT * FROM carts WHERE token = $1;

-- One statement, so two first requests with the same token cannot make two
-- carts: the unique index sends the loser to the DO UPDATE, which changes
-- nothing but lets RETURNING hand back the winner's row.
-- name: UpsertCartByToken :one
INSERT INTO carts (token) VALUES ($1)
ON CONFLICT (token) DO UPDATE SET token = EXCLUDED.token
RETURNING *;
