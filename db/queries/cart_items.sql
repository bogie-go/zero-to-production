-- name: GetCartItem :one
SELECT * FROM cart_items WHERE id = $1;

-- name: ListCartItems :many
SELECT * FROM cart_items ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateCartItem :one
INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateCartItem :one
UPDATE cart_items SET cart_id = $2, product_id = $3, quantity = $4, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteCartItem :execrows
DELETE FROM cart_items WHERE id = $1;
