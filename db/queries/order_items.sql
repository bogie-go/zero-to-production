-- Written by `bogie g model`: overwritten whole on the next run. A
-- hand-written query goes in order_items_extra.sql, not here.

-- name: GetOrderItem :one
SELECT * FROM order_items WHERE id = $1;

-- name: ListOrderItems :many
SELECT * FROM order_items ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateOrderItem :one
UPDATE order_items SET order_id = $2, product_id = $3, quantity = $4, unit_price_cents = $5, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteOrderItem :execrows
DELETE FROM order_items WHERE id = $1;
