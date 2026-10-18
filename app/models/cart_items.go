package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// CartItems, as the domain sees them. Each method converts domain.CartItem to and from
// the row type sqlc generated, and maps pgx's "no rows" to domain.ErrNotFound
// so callers never import pgx to ask a question.

// CreateCartItem validates a cart_item, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateCartItem(ctx context.Context, c domain.CartItem) (domain.CartItem, error) {
	if err := c.Validate(); err != nil {
		return domain.CartItem{}, err
	}
	row, err := s.q.CreateCartItem(ctx, CreateCartItemParams{CartID: c.CartID, ProductID: c.ProductID, Quantity: c.Quantity})
	if err != nil {
		return domain.CartItem{}, fmt.Errorf("models: create cart_item: %w", err)
	}
	return cart_itemFromRow(row), nil
}

// GetCartItem finds one cart_item. An id that is not a valid id is simply not found.
func (s *Store) GetCartItem(ctx context.Context, id string) (domain.CartItem, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.CartItem{}, domain.ErrNotFound
	}
	row, err := s.q.GetCartItem(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.CartItem{}, domain.ErrNotFound
	case err != nil:
		return domain.CartItem{}, fmt.Errorf("models: get cart_item %s: %w", id, err)
	}
	return cart_itemFromRow(row), nil
}

// ListCartItems returns the newest cart_items first, at most limit of them.
func (s *Store) ListCartItems(ctx context.Context, limit int32) ([]domain.CartItem, error) {
	rows, err := s.q.ListCartItems(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("models: list cart_items: %w", err)
	}
	out := make([]domain.CartItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, cart_itemFromRow(r))
	}
	return out, nil
}

// UpdateCartItem validates a cart_item, writes its columns, and returns the stored row.
func (s *Store) UpdateCartItem(ctx context.Context, c domain.CartItem) (domain.CartItem, error) {
	if err := c.Validate(); err != nil {
		return domain.CartItem{}, err
	}
	pk := c.ID
	row, err := s.q.UpdateCartItem(ctx, UpdateCartItemParams{ID: pk, CartID: c.CartID, ProductID: c.ProductID, Quantity: c.Quantity})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.CartItem{}, domain.ErrNotFound
	case err != nil:
		return domain.CartItem{}, fmt.Errorf("models: update cart_item %v: %w", c.ID, err)
	}
	return cart_itemFromRow(row), nil
}

// DeleteCartItem removes a cart_item, and says so if there was none.
func (s *Store) DeleteCartItem(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return domain.ErrNotFound
	}
	n, err := s.q.DeleteCartItem(ctx, pk)
	if err != nil {
		return fmt.Errorf("models: delete cart_item %s: %w", id, err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func cart_itemFromRow(r CartItem) domain.CartItem {
	return domain.CartItem{
		ID:        r.ID,
		CartID:    r.CartID,
		ProductID: r.ProductID,
		Quantity:  r.Quantity,
		CreatedAt: r.CreatedAt.Time,
		UpdatedAt: r.UpdatedAt.Time,
	}
}
