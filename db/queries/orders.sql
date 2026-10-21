-- Written by `bogie g model`: overwritten whole on the next run. A
-- hand-written query goes in orders_extra.sql, not here.

-- name: GetOrder :one
SELECT * FROM orders WHERE id = $1;

-- name: ListOrders :many
SELECT * FROM orders ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateOrder :one
INSERT INTO orders (user_id, status, subtotal_cents, discount_cents, total_cents) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateOrder :one
UPDATE orders SET user_id = $2, status = $3, subtotal_cents = $4, discount_cents = $5, total_cents = $6, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteOrder :execrows
DELETE FROM orders WHERE id = $1;
