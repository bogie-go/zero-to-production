-- +goose Up
CREATE TABLE cart_items (
    id         bigserial   PRIMARY KEY,
    cart_id    bigint      NOT NULL REFERENCES carts (id),
    product_id bigint      NOT NULL REFERENCES products (id),
    quantity   integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cart_items_cart_id_idx ON cart_items (cart_id);
CREATE INDEX cart_items_product_id_idx ON cart_items (product_id);

-- +goose Down
DROP TABLE cart_items;
