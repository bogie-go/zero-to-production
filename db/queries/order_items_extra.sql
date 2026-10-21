-- name: ListOrderItemsForOrder :many
SELECT order_items.id, order_items.product_id, order_items.quantity, order_items.unit_price_cents,
       products.name, products.image_url
FROM order_items
JOIN products ON products.id = order_items.product_id
WHERE order_items.order_id = $1
ORDER BY order_items.created_at;
