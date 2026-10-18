-- name: GetCartByToken :one
SELECT * FROM carts WHERE token = $1;
