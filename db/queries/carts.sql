-- name: GetCart :one
SELECT * FROM carts WHERE id = $1;

-- name: ListCarts :many
SELECT * FROM carts ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateCart :one
INSERT INTO carts (token) VALUES ($1) RETURNING *;

-- name: UpdateCart :one
UPDATE carts SET token = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteCart :execrows
DELETE FROM carts WHERE id = $1;
