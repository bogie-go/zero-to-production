package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// FindOrCreateCartByToken returns the cart with the given token, creating
// an empty one if none exists yet. Unlike Rails's find_or_create_by this is
// atomic: one INSERT ... ON CONFLICT against the unique index on token, so
// two simultaneous first requests end up with the same cart.
//
// The token is whatever identifies "this visitor's cart": a guest's id or an
// account's user id.
func (s *Store) FindOrCreateCartByToken(ctx context.Context, token string) (domain.Cart, error) {
	row, err := s.q.UpsertCartByToken(ctx, token)
	if err != nil {
		return domain.Cart{}, fmt.Errorf("models: find or create cart by token: %w", err)
	}
	return cartFromRow(row), nil
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
// to keep. cart_items has no ON DELETE CASCADE, so its lines are deleted
// first; both deletes run in one transaction (withTx, added for Task E's
// checkout), so a failure midway never leaves a cart with no items but no
// row deleted, or the reverse.
func (s *Store) DeleteCartAndItems(ctx context.Context, cartID int64) error {
	return s.withTx(ctx, func(q *Queries) error {
		if _, err := q.DeleteCartItemsForCart(ctx, cartID); err != nil {
			return fmt.Errorf("models: delete cart items for cart %d: %w", cartID, err)
		}
		n, err := q.DeleteCart(ctx, cartID)
		if err != nil {
			return fmt.Errorf("models: delete cart %d: %w", cartID, err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
