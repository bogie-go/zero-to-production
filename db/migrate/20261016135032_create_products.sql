-- +goose Up
CREATE TABLE products (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       varchar     NOT NULL DEFAULT '',
    description text        NOT NULL DEFAULT '',
    price_cents integer     NOT NULL DEFAULT 0,
    stock      integer     NOT NULL DEFAULT 0,
    image_url  varchar     NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE products;
