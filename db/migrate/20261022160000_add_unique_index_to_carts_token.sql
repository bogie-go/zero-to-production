-- +goose Up
-- One cart per token: a guest or an account is its cart's token, so two rows
-- with the same one is a bug, and find-or-create relies on this to be safe.
CREATE UNIQUE INDEX index_carts_on_token ON carts (token);

-- +goose Down
DROP INDEX index_carts_on_token;
