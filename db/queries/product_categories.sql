-- Written by `bogie g model`: overwritten whole on the next run. A
-- hand-written query goes in product_categories_extra.sql, not here.

-- name: GetProductCategory :one
SELECT * FROM product_categories WHERE id = $1;

-- name: ListProductCategories :many
SELECT * FROM product_categories ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateProductCategory :one
INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2) RETURNING *;

-- name: UpdateProductCategory :one
UPDATE product_categories SET product_id = $2, category_id = $3, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteProductCategory :execrows
DELETE FROM product_categories WHERE id = $1;
