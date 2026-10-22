-- name: GetProductByName :one
SELECT * FROM products WHERE name = $1;

-- name: DecrementProductStock :execrows
UPDATE products SET stock = stock - $1, updated_at = now() WHERE id = $2 AND stock >= $1 AND archived_at IS NULL;

-- The shop's catalog: active products, each with its category's name (a LEFT
-- JOIN, so an uncategorized product still appears), optionally one category's.
-- narg makes category_id optional: NULL means every category.
-- name: ListActiveProducts :many
SELECT p.id, p.name, p.description, p.price_cents, p.stock, p.image_url, p.archived_at,
       p.category_id, p.created_at, p.updated_at, c.name AS category_name
FROM products p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.archived_at IS NULL
  AND (sqlc.narg('category_id')::bigint IS NULL OR p.category_id = sqlc.narg('category_id')::bigint)
ORDER BY p.created_at DESC, p.id DESC
LIMIT sqlc.arg('row_limit');

-- name: GetActiveProduct :one
SELECT p.id, p.name, p.description, p.price_cents, p.stock, p.image_url, p.archived_at,
       p.category_id, p.created_at, p.updated_at, c.name AS category_name
FROM products p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.id = $1 AND p.archived_at IS NULL;

-- name: ArchiveProduct :one
UPDATE products SET archived_at = COALESCE(archived_at, now()), updated_at = now() WHERE id = $1 RETURNING *;

-- name: RestoreProduct :one
UPDATE products SET archived_at = NULL, updated_at = now() WHERE id = $1 RETURNING *;
