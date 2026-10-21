-- +goose Up
CREATE TABLE order_items (
    id         bigserial   PRIMARY KEY,
    order_id   bigint      NOT NULL REFERENCES orders (id),
    product_id bigint      NOT NULL REFERENCES products (id),
    quantity   integer     NOT NULL DEFAULT 0,
    unit_price_cents integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX order_items_order_id_idx ON order_items (order_id);
CREATE INDEX order_items_product_id_idx ON order_items (product_id);

-- +goose Down
DROP TABLE order_items;
