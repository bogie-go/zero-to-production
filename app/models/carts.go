package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
	"ecommerce/db/sqlc"
)

// Carts, as the domain sees them. Each method converts domain.Cart to and from
// the row type sqlc generated, and maps pgx's "no rows" to domain.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in carts_extra.go, not here.

// CreateCart validates a cart, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateCart(ctx context.Context, c domain.Cart) (domain.Cart, error) {
	if err := c.Validate(); err != nil {
		return domain.Cart{}, err
	}
	row, err := s.q.CreateCart(ctx, c.Token)
	if err != nil {
		return domain.Cart{}, fmt.Errorf("models: create cart: %w", err)
	}
	return cartFromRow(row), nil
}

// GetCart finds one cart. An id that is not a valid id is simply not found.
func (s *Store) GetCart(ctx context.Context, id string) (domain.Cart, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.Cart{}, domain.ErrNotFound
	}
	row, err := s.q.GetCart(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Cart{}, domain.ErrNotFound
	case err != nil:
		return domain.Cart{}, fmt.Errorf("models: get cart %s: %w", id, err)
	}
	return cartFromRow(row), nil
}

// ListCarts returns the newest carts first, at most limit of them.
func (s *Store) ListCarts(ctx context.Context, limit int32) ([]domain.Cart, error) {
	rows, err := s.q.ListCarts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("models: list carts: %w", err)
	}
	out := make([]domain.Cart, 0, len(rows))
	for _, r := range rows {
		out = append(out, cartFromRow(r))
	}
	return out, nil
}

// UpdateCart validates a cart, writes its columns, and returns the stored row.
func (s *Store) UpdateCart(ctx context.Context, c domain.Cart) (domain.Cart, error) {
	if err := c.Validate(); err != nil {
		return domain.Cart{}, err
	}
	pk := c.ID
	row, err := s.q.UpdateCart(ctx, sqlc.UpdateCartParams{ID: pk, Token: c.Token})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Cart{}, domain.ErrNotFound
	case err != nil:
		return domain.Cart{}, fmt.Errorf("models: update cart %v: %w", c.ID, err)
	}
	return cartFromRow(row), nil
}

// DeleteCart removes a cart, and says so if there was none.
func (s *Store) DeleteCart(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return domain.ErrNotFound
	}
	n, err := s.q.DeleteCart(ctx, pk)
	if err != nil {
		return fmt.Errorf("models: delete cart %s: %w", id, err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func cartFromRow(r sqlc.Cart) domain.Cart {
	return domain.Cart{
		ID:        r.ID,
		Token:     r.Token,
		CreatedAt: r.CreatedAt.Time,
		UpdatedAt: r.UpdatedAt.Time,
	}
}
