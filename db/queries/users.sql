-- Written by `bogie g model`: overwritten whole on the next run. A
-- hand-written query goes in users_extra.sql, not here.

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: CreateUser :one
INSERT INTO users (email, password_digest, role) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateUser :one
UPDATE users SET email = $2, password_digest = $3, role = $4, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;
