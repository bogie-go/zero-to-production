-- +goose Up
ALTER TABLE products ADD COLUMN archived_at timestamptz;

-- +goose Down
ALTER TABLE products DROP COLUMN archived_at;
