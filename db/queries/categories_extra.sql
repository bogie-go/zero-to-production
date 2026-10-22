-- Every category with how many ACTIVE products it holds: a LEFT JOIN with an
-- aggregate, so an empty category appears with a count of 0.
-- name: ListCategoriesWithCounts :many
SELECT c.id, c.name, c.created_at, c.updated_at, COUNT(p.id) AS product_count
FROM categories c
LEFT JOIN products p ON p.category_id = c.id AND p.archived_at IS NULL
GROUP BY c.id
ORDER BY c.name;

-- name: GetCategoryByName :one
SELECT * FROM categories WHERE name = $1;
