-- name: GetPriceRuleByName :one
SELECT * FROM price_rules WHERE name = $1;

-- name: ListActivePriceRules :many
SELECT * FROM price_rules
WHERE active
  AND (starts_at IS NULL OR starts_at <= now())
  AND (ends_at IS NULL OR ends_at >= now())
ORDER BY priority, created_at;
