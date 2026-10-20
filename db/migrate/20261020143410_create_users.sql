-- +goose Up
CREATE TABLE users (
    id         bigserial   PRIMARY KEY,
    email      varchar     NOT NULL DEFAULT '' UNIQUE,
    password_digest varchar     NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
