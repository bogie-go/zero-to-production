package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/models"
	"ecommerce/app/pricing"
	"ecommerce/db/sqlc"
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
func (s *Store) CreateOrderFromCart(ctx context.Context, userID int64, cartID int64) (models.Order, []models.OrderLine, error) {
	lines, err := s.CartContents(ctx, cartID)
	if err != nil {
		return models.Order{}, nil, err
	}
	if len(lines) == 0 {
		return models.Order{}, nil, fmt.Errorf("%w: cart is empty", models.ErrInvalid)
	}

	rules, err := s.ListActivePriceRules(ctx)
	if err != nil {
		return models.Order{}, nil, err
	}
	var subtotal int32
	for _, l := range lines {
		subtotal += l.UnitPriceCents * l.Quantity
	}
	result, err := pricing.CalculateLines(pricing.CartLines(lines), rules)
	if err != nil {
		return models.Order{}, nil, err
	}

	var order models.Order
	err = s.withTx(ctx, func(tx pgx.Tx, q *sqlc.Queries) error {
		for _, l := range lines {
			n, err := q.DecrementProductStock(ctx, sqlc.DecrementProductStockParams{Stock: l.Quantity, ID: l.ProductID})
			if err != nil {
				return fmt.Errorf("store: decrement stock for product %d: %w", l.ProductID, err)
			}
			if n == 0 {
				return fmt.Errorf("%w: %s is no longer available in that quantity", models.ErrInvalid, l.ProductName)
			}
		}

		o := models.Order{
			UserID:        userID,
			Status:        "pending",
			SubtotalCents: subtotal,
			DiscountCents: int32(result.TotalSavingsCents()),
			TotalCents:    int32(result.FinalCents),
		}
		if err := o.Validate(); err != nil {
			return err
		}
		row, err := q.CreateOrder(ctx, sqlc.CreateOrderParams{
			UserID: o.UserID, Status: o.Status,
			SubtotalCents: o.SubtotalCents, DiscountCents: o.DiscountCents, TotalCents: o.TotalCents,
		})
		if err != nil {
			return fmt.Errorf("store: create order: %w", err)
		}
		order = orderFromRow(row)

		for _, l := range lines {
			item := models.OrderItem{
				OrderID: order.ID, ProductID: l.ProductID,
				Quantity: l.Quantity, UnitPriceCents: l.UnitPriceCents,
			}
			if err := item.Validate(); err != nil {
				return err
			}
			if _, err := q.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
				OrderID: item.OrderID, ProductID: item.ProductID,
				Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents,
			}); err != nil {
				return fmt.Errorf("store: create order item: %w", err)
			}
		}

		if _, err := q.DeleteCartItemsForCart(ctx, cartID); err != nil {
			return fmt.Errorf("store: clear cart %d: %w", cartID, err)
		}

		// Last, still inside the transaction: whatever the hook does (enqueue
		// the confirmation email) commits with the order or not at all, and a
		// hook that fails rolls the whole checkout back.
		if s.onOrderPlaced != nil {
			if err := s.onOrderPlaced(ctx, tx, order); err != nil {
				return fmt.Errorf("store: after order %d: %w", order.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return models.Order{}, nil, err
	}

	orderLines := make([]models.OrderLine, 0, len(lines))
	for _, l := range lines {
		orderLines = append(orderLines, models.OrderLine{
			ProductID: l.ProductID, ProductName: l.ProductName, ImageUrl: l.ImageUrl,
			UnitPriceCents: l.UnitPriceCents, Quantity: l.Quantity,
		})
	}
	return order, orderLines, nil
}

// OnOrderPlaced registers a function CreateOrderFromCart calls inside its
// transaction, after the order is written, with that transaction. It exists
// so the app can enqueue a job in the same transaction (River's InsertTx)
// without this package knowing about River: the job is created if and only if
// the order is. app/application.go sets it; tests that need no job leave it
// unset.
func (s *Store) OnOrderPlaced(fn func(ctx context.Context, tx pgx.Tx, order models.Order) error) {
	s.onOrderPlaced = fn
}

// ListOrdersForUser returns a user's own orders, newest first — never
// another user's, since every caller here names the user from their own
// token (middlewares.Subject), never an id from the URL.
func (s *Store) ListOrdersForUser(ctx context.Context, userID int64, limit int32) ([]models.Order, error) {
	rows, err := s.q.ListOrdersForUser(ctx, sqlc.ListOrdersForUserParams{UserID: userID, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("store: list orders for user %d: %w", userID, err)
	}
	out := make([]models.Order, 0, len(rows))
	for _, r := range rows {
		out = append(out, orderFromRow(r))
	}
	return out, nil
}

// GetOrderForUser finds one order and its lines, but only if it belongs to
// userID — the same ownership scoping cart items use: a caller cannot tell
// "someone else's order" from "no such order" apart, id guessing included.
func (s *Store) GetOrderForUser(ctx context.Context, userID int64, id string) (models.Order, []models.OrderLine, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.Order{}, nil, models.ErrNotFound
	}
	row, err := s.q.GetOrderForUser(ctx, sqlc.GetOrderForUserParams{ID: pk, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Order{}, nil, models.ErrNotFound
	case err != nil:
		return models.Order{}, nil, fmt.Errorf("store: get order %s for user %d: %w", id, userID, err)
	}
	order := orderFromRow(row)

	rows, err := s.q.ListOrderItemsForOrder(ctx, order.ID)
	if err != nil {
		return models.Order{}, nil, fmt.Errorf("store: list order items for order %d: %w", order.ID, err)
	}
	lines := make([]models.OrderLine, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, models.OrderLine{
			ProductID: r.ProductID, ProductName: r.Name, ImageUrl: r.ImageUrl,
			UnitPriceCents: r.UnitPriceCents, Quantity: r.Quantity,
		})
	}
	return order, lines, nil
}
