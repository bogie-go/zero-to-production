package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// Orders, as the domain sees them. Each method converts domain.Order to and from
// the row type sqlc generated, and maps pgx's "no rows" to domain.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in orders_extra.go, not here.

// CreateOrder validates a order, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateOrder(ctx context.Context, o domain.Order) (domain.Order, error) {
	if err := o.Validate(); err != nil {
		return domain.Order{}, err
	}
	row, err := s.q.CreateOrder(ctx, CreateOrderParams{UserID: o.UserID, Status: o.Status, SubtotalCents: o.SubtotalCents, DiscountCents: o.DiscountCents, TotalCents: o.TotalCents})
	if err != nil {
		return domain.Order{}, fmt.Errorf("models: create order: %w", err)
	}
	return orderFromRow(row), nil
}

// GetOrder finds one order. An id that is not a valid id is simply not found.
func (s *Store) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.Order{}, domain.ErrNotFound
	}
	row, err := s.q.GetOrder(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Order{}, domain.ErrNotFound
	case err != nil:
		return domain.Order{}, fmt.Errorf("models: get order %s: %w", id, err)
	}
	return orderFromRow(row), nil
}

// ListOrders returns the newest orders first, at most limit of them.
func (s *Store) ListOrders(ctx context.Context, limit int32) ([]domain.Order, error) {
	rows, err := s.q.ListOrders(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("models: list orders: %w", err)
	}
	out := make([]domain.Order, 0, len(rows))
	for _, r := range rows {
		out = append(out, orderFromRow(r))
	}
	return out, nil
}

// UpdateOrder validates a order, writes its columns, and returns the stored row.
func (s *Store) UpdateOrder(ctx context.Context, o domain.Order) (domain.Order, error) {
	if err := o.Validate(); err != nil {
		return domain.Order{}, err
	}
	pk := o.ID
	row, err := s.q.UpdateOrder(ctx, UpdateOrderParams{ID: pk, UserID: o.UserID, Status: o.Status, SubtotalCents: o.SubtotalCents, DiscountCents: o.DiscountCents, TotalCents: o.TotalCents})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Order{}, domain.ErrNotFound
	case err != nil:
		return domain.Order{}, fmt.Errorf("models: update order %v: %w", o.ID, err)
	}
	return orderFromRow(row), nil
}

// DeleteOrder removes a order, and says so if there was none.
func (s *Store) DeleteOrder(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return domain.ErrNotFound
	}
	n, err := s.q.DeleteOrder(ctx, pk)
	if err != nil {
		return fmt.Errorf("models: delete order %s: %w", id, err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func orderFromRow(r Order) domain.Order {
	return domain.Order{
		ID:            r.ID,
		UserID:        r.UserID,
		Status:        r.Status,
		SubtotalCents: r.SubtotalCents,
		DiscountCents: r.DiscountCents,
		TotalCents:    r.TotalCents,
		CreatedAt:     r.CreatedAt.Time,
		UpdatedAt:     r.UpdatedAt.Time,
	}
}
