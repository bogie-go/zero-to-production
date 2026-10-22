-- +goose Up
CREATE TABLE categories (
    id         bigserial   PRIMARY KEY,
    name       varchar     NOT NULL DEFAULT '' UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE categories;
