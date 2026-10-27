package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/models"
	"ecommerce/db/sqlc"
)

// Carts, as the models see them. Each method converts models.Cart to and from
// the row type sqlc generated, and maps pgx's "no rows" to models.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in carts_extra.go, not here.

// CreateCart validates a cart, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateCart(ctx context.Context, c models.Cart) (models.Cart, error) {
	if err := c.Validate(); err != nil {
		return models.Cart{}, err
	}
	row, err := s.q.CreateCart(ctx, c.Token)
	if err != nil {
		return models.Cart{}, fmt.Errorf("store: create cart: %w", err)
	}
	return cartFromRow(row), nil
}

// GetCart finds one cart. An id that is not a valid id is simply not found.
func (s *Store) GetCart(ctx context.Context, id string) (models.Cart, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.Cart{}, models.ErrNotFound
	}
	row, err := s.q.GetCart(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Cart{}, models.ErrNotFound
	case err != nil:
		return models.Cart{}, fmt.Errorf("store: get cart %s: %w", id, err)
	}
	return cartFromRow(row), nil
}

// ListCarts returns the newest carts first, at most limit of them.
func (s *Store) ListCarts(ctx context.Context, limit int32) ([]models.Cart, error) {
	rows, err := s.q.ListCarts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list carts: %w", err)
	}
	out := make([]models.Cart, 0, len(rows))
	for _, r := range rows {
		out = append(out, cartFromRow(r))
	}
	return out, nil
}

// UpdateCart validates a cart, writes its columns, and returns the stored row.
func (s *Store) UpdateCart(ctx context.Context, c models.Cart) (models.Cart, error) {
	if err := c.Validate(); err != nil {
		return models.Cart{}, err
	}
	pk := c.ID
	row, err := s.q.UpdateCart(ctx, sqlc.UpdateCartParams{ID: pk, Token: c.Token})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Cart{}, models.ErrNotFound
	case err != nil:
		return models.Cart{}, fmt.Errorf("store: update cart %v: %w", c.ID, err)
	}
	return cartFromRow(row), nil
}

// DeleteCart removes a cart, and says so if there was none.
func (s *Store) DeleteCart(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return models.ErrNotFound
	}
	n, err := s.q.DeleteCart(ctx, pk)
	if err != nil {
		return fmt.Errorf("store: delete cart %s: %w", id, err)
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func cartFromRow(r sqlc.Cart) models.Cart {
	return models.Cart{
		ID:        r.ID,
		Token:     r.Token,
		CreatedAt: r.CreatedAt.Time,
		UpdatedAt: r.UpdatedAt.Time,
	}
}
