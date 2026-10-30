-- Every category with how many ACTIVE products it holds. Both joins are LEFT,
-- so an empty category still appears with a count of 0 and a membership of an
-- archived product only fails to count. The archive check is in ON, not in a
-- WHERE: a WHERE would drop the category whose only product is archived.
-- name: ListCategoriesWithCounts :many
SELECT c.id, c.name, c.created_at, c.updated_at, COUNT(p.id) AS product_count
FROM categories c
LEFT JOIN product_categories pc ON pc.category_id = c.id
LEFT JOIN products p ON p.id = pc.product_id AND p.archived_at IS NULL
GROUP BY c.id
ORDER BY c.name;

-- The categories one product is in: the "through" of a many-to-many.
-- name: ListCategoriesForProduct :many
SELECT c.* FROM categories c
JOIN product_categories pc ON pc.category_id = c.id
WHERE pc.product_id = $1
ORDER BY c.name;

-- name: GetCategoryByName :one
SELECT * FROM categories WHERE name = $1;
