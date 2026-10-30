-- +goose Up
CREATE TABLE product_categories (
    id          bigserial   PRIMARY KEY,
    product_id  bigint      NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    category_id bigint      NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX product_categories_product_id_idx ON product_categories (product_id);
CREATE INDEX product_categories_category_id_idx ON product_categories (category_id);
CREATE UNIQUE INDEX product_categories_product_id_category_id_idx ON product_categories (product_id, category_id);

-- +goose Down
DROP TABLE product_categories;
