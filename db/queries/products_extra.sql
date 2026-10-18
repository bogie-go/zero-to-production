-- name: GetProductByName :one
SELECT * FROM products WHERE name = $1;
