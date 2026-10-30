-- name: GetProductByName :one
SELECT * FROM products WHERE name = $1;

-- name: DecrementProductStock :execrows
UPDATE products SET stock = stock - $1, updated_at = now() WHERE id = $2 AND stock >= $1 AND archived_at IS NULL;

-- The shop's catalog: active products, optionally one category's. narg makes
-- category_id optional: NULL means every category. The filter is an EXISTS,
-- not a JOIN: a product in several categories must still appear once.
-- name: ListActiveProducts :many
SELECT * FROM products p
WHERE p.archived_at IS NULL
  AND (sqlc.narg('category_id')::bigint IS NULL OR EXISTS (
        SELECT 1 FROM product_categories pc
        WHERE pc.product_id = p.id AND pc.category_id = sqlc.narg('category_id')::bigint))
ORDER BY p.created_at DESC, p.id DESC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- How many products that list holds in all, for the pager. The same filter,
-- so the count and the pages can never disagree.
-- name: CountActiveProducts :one
SELECT count(*) FROM products p
WHERE p.archived_at IS NULL
  AND (sqlc.narg('category_id')::bigint IS NULL OR EXISTS (
        SELECT 1 FROM product_categories pc
        WHERE pc.product_id = p.id AND pc.category_id = sqlc.narg('category_id')::bigint));

-- name: GetActiveProduct :one
SELECT * FROM products WHERE id = $1 AND archived_at IS NULL;

-- Locks the product's row until the transaction ends, so two writers of its
-- categories take turns.
-- name: GetProductForUpdate :one
SELECT * FROM products WHERE id = $1 FOR UPDATE;

-- name: ArchiveProduct :one
UPDATE products SET archived_at = COALESCE(archived_at, now()), updated_at = now() WHERE id = $1 RETURNING *;

-- name: RestoreProduct :one
UPDATE products SET archived_at = NULL, updated_at = now() WHERE id = $1 RETURNING *;

-- The admin's list: every product, or only the unarchived, a page at a time.
-- The order is total (created_at, then id), so pages never overlap or skip.
-- name: ListProductsPage :many
SELECT * FROM products
WHERE (sqlc.arg('include_archived')::boolean OR archived_at IS NULL)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountProducts :one
SELECT count(*) FROM products
WHERE (sqlc.arg('include_archived')::boolean OR archived_at IS NULL);
