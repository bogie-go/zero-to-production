-- Written by `bogie g model`: overwritten whole on the next run. A
-- hand-written query goes in products_extra.sql, not here.

-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: ListProducts :many
SELECT * FROM products ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateProduct :one
INSERT INTO products (name, description, price_cents, stock, image_url) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateProduct :one
UPDATE products SET name = $2, description = $3, price_cents = $4, stock = $5, image_url = $6, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteProduct :execrows
DELETE FROM products WHERE id = $1;
