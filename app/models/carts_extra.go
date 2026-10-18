package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// FindOrCreateCartByToken returns the cart with the given token, creating
// an empty one if none exists yet. Like Rails's find_or_create_by, this is
// a plain find then create, not atomic: a race between two callers can
// still create two carts with the same token.
//
// The token is whatever identifies "this visitor's cart": the empty string
// for the single shared cart task C starts with, a per-visitor token once
// one is issued.
func (s *Store) FindOrCreateCartByToken(ctx context.Context, token string) (domain.Cart, error) {
	row, err := s.q.GetCartByToken(ctx, token)
	switch {
	case err == nil:
		return cartFromRow(row), nil
	case errors.Is(err, pgx.ErrNoRows):
		return s.CreateCart(ctx, domain.Cart{Token: token})
	default:
		return domain.Cart{}, fmt.Errorf("models: get cart by token: %w", err)
	}
}
