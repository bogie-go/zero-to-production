-- Written by `bogie g model`: overwritten whole on the next run. A
-- hand-written query goes in categories_extra.sql, not here.

-- name: GetCategory :one
SELECT * FROM categories WHERE id = $1;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateCategory :one
INSERT INTO categories (name) VALUES ($1) RETURNING *;

-- name: UpdateCategory :one
UPDATE categories SET name = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM categories WHERE id = $1;
