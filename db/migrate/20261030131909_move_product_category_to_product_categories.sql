-- A product is in many categories now: its one category becomes a row of
-- product_categories, and products.category_id goes.

-- +goose Up
INSERT INTO product_categories (product_id, category_id)
SELECT id, category_id FROM products WHERE category_id IS NOT NULL;
ALTER TABLE products DROP COLUMN category_id;

-- +goose Down
ALTER TABLE products ADD COLUMN category_id bigint REFERENCES categories (id) ON DELETE SET NULL;
CREATE INDEX products_category_id_idx ON products (category_id);
UPDATE products SET category_id = (
    SELECT category_id FROM product_categories
    WHERE product_id = products.id ORDER BY id LIMIT 1);
