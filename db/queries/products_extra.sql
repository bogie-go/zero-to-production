-- name: GetProductByName :one
SELECT * FROM products WHERE name = $1;

-- name: DecrementProductStock :execrows
UPDATE products SET stock = stock - $1, updated_at = now() WHERE id = $2 AND stock >= $1;
