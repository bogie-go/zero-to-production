-- +goose Up
CREATE TABLE orders (
    id         bigserial   PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id),
    status     varchar     NOT NULL DEFAULT '',
    subtotal_cents integer     NOT NULL DEFAULT 0,
    discount_cents integer     NOT NULL DEFAULT 0,
    total_cents integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_user_id_idx ON orders (user_id);

-- +goose Down
DROP TABLE orders;
