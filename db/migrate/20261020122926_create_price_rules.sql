-- +goose Up
CREATE TABLE price_rules (
    id         bigserial   PRIMARY KEY,
    name       varchar     NOT NULL DEFAULT '',
    kind       varchar     NOT NULL DEFAULT '',
    properties jsonb       NOT NULL DEFAULT '{}',
    active     boolean     NOT NULL DEFAULT true,
    priority   integer     NOT NULL DEFAULT 0,
    starts_at  timestamptz,
    ends_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE price_rules;
