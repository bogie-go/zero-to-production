-- name: DeleteProductCategoriesForProduct :exec
DELETE FROM product_categories WHERE product_id = $1;
