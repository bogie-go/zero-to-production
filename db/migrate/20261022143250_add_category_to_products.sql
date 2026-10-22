-- +goose Up
ALTER TABLE products ADD COLUMN category_id bigint REFERENCES categories (id) ON DELETE SET NULL;
CREATE INDEX products_category_id_idx ON products (category_id);

-- +goose Down
ALTER TABLE products DROP COLUMN category_id;
