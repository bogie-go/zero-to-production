package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/models"
)

// OrderItems, as the models see them. Each method converts models.OrderItem to and from
// the row type sqlc generated, and maps pgx's "no rows" to models.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in order_items_extra.go, not here.

// CreateOrderItem validates a order_item, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateOrderItem(ctx context.Context, o models.OrderItem) (models.OrderItem, error) {
	if err := o.Validate(); err != nil {
		return models.OrderItem{}, err
	}
	row, err := s.q.CreateOrderItem(ctx, CreateOrderItemParams{OrderID: o.OrderID, ProductID: o.ProductID, Quantity: o.Quantity, UnitPriceCents: o.UnitPriceCents})
	if err != nil {
		return models.OrderItem{}, fmt.Errorf("store: create order_item: %w", err)
	}
	return order_itemFromRow(row), nil
}

// GetOrderItem finds one order_item. An id that is not a valid id is simply not found.
func (s *Store) GetOrderItem(ctx context.Context, id string) (models.OrderItem, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.OrderItem{}, models.ErrNotFound
	}
	row, err := s.q.GetOrderItem(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.OrderItem{}, models.ErrNotFound
	case err != nil:
		return models.OrderItem{}, fmt.Errorf("store: get order_item %s: %w", id, err)
	}
	return order_itemFromRow(row), nil
}

// ListOrderItems returns the newest order_items first, at most limit of them.
func (s *Store) ListOrderItems(ctx context.Context, limit int32) ([]models.OrderItem, error) {
	rows, err := s.q.ListOrderItems(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list order_items: %w", err)
	}
	out := make([]models.OrderItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, order_itemFromRow(r))
	}
	return out, nil
}

// UpdateOrderItem validates a order_item, writes its columns, and returns the stored row.
func (s *Store) UpdateOrderItem(ctx context.Context, o models.OrderItem) (models.OrderItem, error) {
	if err := o.Validate(); err != nil {
		return models.OrderItem{}, err
	}
	pk := o.ID
	row, err := s.q.UpdateOrderItem(ctx, UpdateOrderItemParams{ID: pk, OrderID: o.OrderID, ProductID: o.ProductID, Quantity: o.Quantity, UnitPriceCents: o.UnitPriceCents})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.OrderItem{}, models.ErrNotFound
	case err != nil:
		return models.OrderItem{}, fmt.Errorf("store: update order_item %v: %w", o.ID, err)
	}
	return order_itemFromRow(row), nil
}

// DeleteOrderItem removes a order_item, and says so if there was none.
func (s *Store) DeleteOrderItem(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return models.ErrNotFound
	}
	n, err := s.q.DeleteOrderItem(ctx, pk)
	if err != nil {
		return fmt.Errorf("store: delete order_item %s: %w", id, err)
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func order_itemFromRow(r OrderItem) models.OrderItem {
	return models.OrderItem{
		ID:             r.ID,
		OrderID:        r.OrderID,
		ProductID:      r.ProductID,
		Quantity:       r.Quantity,
		UnitPriceCents: r.UnitPriceCents,
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
	}
}
