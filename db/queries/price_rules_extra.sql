-- name: GetPriceRuleByName :one
SELECT * FROM price_rules WHERE name = $1;

-- name: ListActivePriceRules :many
SELECT * FROM price_rules
WHERE active
  AND (starts_at IS NULL OR starts_at <= now())
  AND (ends_at IS NULL OR ends_at >= now())
ORDER BY priority, created_at;

-- The admin's list: every rule in the order the engine runs them (priority,
-- then id), active or not.
-- name: ListAllPriceRules :many
SELECT * FROM price_rules ORDER BY priority ASC, id ASC;

-- name: SetPriceRuleActive :one
UPDATE price_rules SET active = $2, updated_at = now() WHERE id = $1 RETURNING *;
