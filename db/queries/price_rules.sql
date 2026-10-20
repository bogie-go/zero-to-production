-- Written by `bogie g model`: overwritten whole on the next run. A
-- hand-written query goes in price_rules_extra.sql, not here.

-- name: GetPriceRule :one
SELECT * FROM price_rules WHERE id = $1;

-- name: ListPriceRules :many
SELECT * FROM price_rules ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreatePriceRule :one
INSERT INTO price_rules (name, kind, properties, active, priority, starts_at, ends_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: UpdatePriceRule :one
UPDATE price_rules SET name = $2, kind = $3, properties = $4, active = $5, priority = $6, starts_at = $7, ends_at = $8, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeletePriceRule :execrows
DELETE FROM price_rules WHERE id = $1;
