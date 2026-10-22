-- name: GetProductByName :one
SELECT * FROM products WHERE name = $1;

-- name: DecrementProductStock :execrows
UPDATE products SET stock = stock - $1, updated_at = now() WHERE id = $2 AND stock >= $1 AND archived_at IS NULL;

-- name: ListActiveProducts :many
SELECT * FROM products WHERE archived_at IS NULL ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: GetActiveProduct :one
SELECT * FROM products WHERE id = $1 AND archived_at IS NULL;

-- name: ArchiveProduct :one
UPDATE products SET archived_at = COALESCE(archived_at, now()), updated_at = now() WHERE id = $1 RETURNING *;

-- name: RestoreProduct :one
UPDATE products SET archived_at = NULL, updated_at = now() WHERE id = $1 RETURNING *;
