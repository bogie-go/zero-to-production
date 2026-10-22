-- +goose Up
ALTER TABLE users ADD COLUMN role varchar NOT NULL DEFAULT 'customer';

-- +goose Down
ALTER TABLE users DROP COLUMN role;
