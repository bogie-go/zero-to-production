package models

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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

// GetCartByToken finds the cart for token without creating one if there is
// none — unlike FindOrCreateCartByToken, for a caller (a login's guest-cart
// merge) that only wants to act on a cart that already exists.
func (s *Store) GetCartByToken(ctx context.Context, token string) (domain.Cart, error) {
	row, err := s.q.GetCartByToken(ctx, token)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Cart{}, domain.ErrNotFound
	case err != nil:
		return domain.Cart{}, fmt.Errorf("models: get cart by token: %w", err)
	}
	return cartFromRow(row), nil
}

// DeleteCartAndItems removes a cart and every line in it — used once a
// guest's cart has been merged into a signed-in user's and has nothing left
// to keep. cart_items has no ON DELETE CASCADE yet (real transactions
// arrive in Task E), so its lines are deleted first.
func (s *Store) DeleteCartAndItems(ctx context.Context, cartID int64) error {
	lines, err := s.CartContents(ctx, cartID)
	if err != nil {
		return err
	}
	for _, l := range lines {
		if err := s.DeleteCartItemForCart(ctx, cartID, strconv.FormatInt(l.ID, 10)); err != nil {
			return err
		}
	}
	return s.DeleteCart(ctx, strconv.FormatInt(cartID, 10))
}
