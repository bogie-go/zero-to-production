package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
	"ecommerce/app/pricing"
)

// CreateOrderFromCart turns a cart into an order: one database transaction
// that, for every line, conditionally decrements the product's stock (the
// same "two shoppers, one last item" race a flash sale creates), then
// writes the order and its order_items with today's prices snapshotted in,
// then empties the cart — or, if any line can't be filled, undoes all of
// it, leaving stock, cart and order table exactly as they were.
//
// userID is the signed-in user placing the order; a guest token never
// resolves to one (see OrdersController), so every order has a real owner.
func (s *Store) CreateOrderFromCart(ctx context.Context, userID int64, cartID int64) (domain.Order, []domain.OrderLine, error) {
	lines, err := s.CartContents(ctx, cartID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if len(lines) == 0 {
		return domain.Order{}, nil, fmt.Errorf("%w: cart is empty", domain.ErrInvalid)
	}

	rules, err := s.ListActivePriceRules(ctx)
	if err != nil {
		return domain.Order{}, nil, err
	}
	var subtotal int32
	for _, l := range lines {
		subtotal += l.UnitPriceCents * l.Quantity
	}
	result, err := pricing.Calculate(int64(subtotal), rules, nil)
	if err != nil {
		return domain.Order{}, nil, err
	}

	var order domain.Order
	err = s.withTx(ctx, func(q *Queries) error {
		for _, l := range lines {
			n, err := q.DecrementProductStock(ctx, DecrementProductStockParams{Stock: l.Quantity, ID: l.ProductID})
			if err != nil {
				return fmt.Errorf("models: decrement stock for product %d: %w", l.ProductID, err)
			}
			if n == 0 {
				return fmt.Errorf("%w: %s is no longer available in that quantity", domain.ErrInvalid, l.ProductName)
			}
		}

		o := domain.Order{
			UserID:        userID,
			Status:        "pending",
			SubtotalCents: subtotal,
			DiscountCents: int32(result.TotalSavingsCents()),
			TotalCents:    int32(result.FinalCents),
		}
		if err := o.Validate(); err != nil {
			return err
		}
		row, err := q.CreateOrder(ctx, CreateOrderParams{
			UserID: o.UserID, Status: o.Status,
			SubtotalCents: o.SubtotalCents, DiscountCents: o.DiscountCents, TotalCents: o.TotalCents,
		})
		if err != nil {
			return fmt.Errorf("models: create order: %w", err)
		}
		order = orderFromRow(row)

		for _, l := range lines {
			item := domain.OrderItem{
				OrderID: order.ID, ProductID: l.ProductID,
				Quantity: l.Quantity, UnitPriceCents: l.UnitPriceCents,
			}
			if err := item.Validate(); err != nil {
				return err
			}
			if _, err := q.CreateOrderItem(ctx, CreateOrderItemParams{
				OrderID: item.OrderID, ProductID: item.ProductID,
				Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents,
			}); err != nil {
				return fmt.Errorf("models: create order item: %w", err)
			}
		}

		if _, err := q.DeleteCartItemsForCart(ctx, cartID); err != nil {
			return fmt.Errorf("models: clear cart %d: %w", cartID, err)
		}
		return nil
	})
	if err != nil {
		return domain.Order{}, nil, err
	}

	orderLines := make([]domain.OrderLine, 0, len(lines))
	for _, l := range lines {
		orderLines = append(orderLines, domain.OrderLine{
			ProductID: l.ProductID, ProductName: l.ProductName, ImageUrl: l.ImageUrl,
			UnitPriceCents: l.UnitPriceCents, Quantity: l.Quantity,
		})
	}
	return order, orderLines, nil
}

// ListOrdersForUser returns a user's own orders, newest first — never
// another user's, since every caller here names the user from their own
// token (middlewares.Subject), never an id from the URL.
func (s *Store) ListOrdersForUser(ctx context.Context, userID int64, limit int32) ([]domain.Order, error) {
	rows, err := s.q.ListOrdersForUser(ctx, ListOrdersForUserParams{UserID: userID, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("models: list orders for user %d: %w", userID, err)
	}
	out := make([]domain.Order, 0, len(rows))
	for _, r := range rows {
		out = append(out, orderFromRow(r))
	}
	return out, nil
}

// GetOrderForUser finds one order and its lines, but only if it belongs to
// userID — the same ownership scoping cart items use: a caller cannot tell
// "someone else's order" from "no such order" apart, id guessing included.
func (s *Store) GetOrderForUser(ctx context.Context, userID int64, id string) (domain.Order, []domain.OrderLine, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.Order{}, nil, domain.ErrNotFound
	}
	row, err := s.q.GetOrderForUser(ctx, GetOrderForUserParams{ID: pk, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Order{}, nil, domain.ErrNotFound
	case err != nil:
		return domain.Order{}, nil, fmt.Errorf("models: get order %s for user %d: %w", id, userID, err)
	}
	order := orderFromRow(row)

	rows, err := s.q.ListOrderItemsForOrder(ctx, order.ID)
	if err != nil {
		return domain.Order{}, nil, fmt.Errorf("models: list order items for order %d: %w", order.ID, err)
	}
	lines := make([]domain.OrderLine, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, domain.OrderLine{
			ProductID: r.ProductID, ProductName: r.Name, ImageUrl: r.ImageUrl,
			UnitPriceCents: r.UnitPriceCents, Quantity: r.Quantity,
		})
	}
	return order, lines, nil
}
