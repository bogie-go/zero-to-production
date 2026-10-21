-- name: GetCartItemByCartAndProduct :one
SELECT * FROM cart_items WHERE cart_id = $1 AND product_id = $2;

-- name: GetCartItemForCart :one
SELECT * FROM cart_items WHERE id = $1 AND cart_id = $2;

-- name: DeleteCartItemForCart :execrows
DELETE FROM cart_items WHERE id = $1 AND cart_id = $2;

-- name: DeleteCartItemsForCart :execrows
DELETE FROM cart_items WHERE cart_id = $1;

-- name: ListCartItemsForCart :many
SELECT cart_items.id, cart_items.product_id, cart_items.quantity,
       products.name, products.image_url, products.price_cents
FROM cart_items
JOIN products ON products.id = cart_items.product_id
WHERE cart_items.cart_id = $1
ORDER BY cart_items.created_at;
