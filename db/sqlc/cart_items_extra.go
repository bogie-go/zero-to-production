package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/models"
)

// AddCartItem adds quantity of a product to a cart, incrementing an
// existing line for that product instead of creating a second one — the
// way adding the same thing to a Rails cart twice usually works. quantity
// must be positive and the line's total, new or incremented, must not
// exceed the product's current stock.
func (s *Store) AddCartItem(ctx context.Context, cartID, productID int64, quantity int32) (models.CartItem, error) {
	if quantity <= 0 {
		return models.CartItem{}, fmt.Errorf("%w: quantity must be positive", models.ErrInvalid)
	}
	product, err := s.q.GetProduct(ctx, productID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.CartItem{}, fmt.Errorf("%w: no such product", models.ErrInvalid)
	case err != nil:
		return models.CartItem{}, fmt.Errorf("store: get product %d: %w", productID, err)
	}
	if product.ArchivedAt.Valid {
		return models.CartItem{}, fmt.Errorf("%w: %s is no longer available", models.ErrInvalid, product.Name)
	}

	existing, err := s.q.GetCartItemByCartAndProduct(ctx, GetCartItemByCartAndProductParams{CartID: cartID, ProductID: productID})
	switch {
	case err == nil:
		quantity += existing.Quantity
		if quantity > product.Stock {
			return models.CartItem{}, fmt.Errorf("%w: only %d in stock", models.ErrInvalid, product.Stock)
		}
		return s.UpdateCartItem(ctx, models.CartItem{ID: existing.ID, CartID: cartID, ProductID: productID, Quantity: quantity})
	case errors.Is(err, pgx.ErrNoRows):
		if quantity > product.Stock {
			return models.CartItem{}, fmt.Errorf("%w: only %d in stock", models.ErrInvalid, product.Stock)
		}
		return s.CreateCartItem(ctx, models.CartItem{CartID: cartID, ProductID: productID, Quantity: quantity})
	default:
		return models.CartItem{}, fmt.Errorf("store: get cart item: %w", err)
	}
}

// UpdateCartItemQuantity sets a line's quantity directly, validating it
// against the product's current stock the same way AddCartItem does. The
// line must belong to cartID — the cart the caller's own token resolved to,
// never a cart id the caller names — or this is ErrNotFound, the same as a
// line that simply doesn't exist: a caller cannot tell "not yours" from
// "doesn't exist" apart, the same way Rails' current_cart.cart_items.find
// would 404 either way. id is a URL-sourced string, like every other store
// method that takes one; an id that is not a valid id is simply not found.
func (s *Store) UpdateCartItemQuantity(ctx context.Context, cartID int64, id string, quantity int32) (models.CartItem, error) {
	if quantity <= 0 {
		return models.CartItem{}, fmt.Errorf("%w: quantity must be positive", models.ErrInvalid)
	}
	pk, err := parseID(id)
	if err != nil {
		return models.CartItem{}, models.ErrNotFound
	}
	row, err := s.q.GetCartItemForCart(ctx, GetCartItemForCartParams{ID: pk, CartID: cartID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.CartItem{}, models.ErrNotFound
	case err != nil:
		return models.CartItem{}, fmt.Errorf("store: get cart item %s for cart %d: %w", id, cartID, err)
	}
	item := cart_itemFromRow(row)
	product, err := s.q.GetProduct(ctx, item.ProductID)
	if err != nil {
		return models.CartItem{}, fmt.Errorf("store: get product %d: %w", item.ProductID, err)
	}
	if product.ArchivedAt.Valid {
		return models.CartItem{}, fmt.Errorf("%w: %s is no longer available", models.ErrInvalid, product.Name)
	}
	if quantity > product.Stock {
		return models.CartItem{}, fmt.Errorf("%w: only %d in stock", models.ErrInvalid, product.Stock)
	}
	item.Quantity = quantity
	return s.UpdateCartItem(ctx, item)
}

// DeleteCartItemForCart removes a line, but only if it belongs to cartID —
// the same ownership check UpdateCartItemQuantity makes, and for the same
// reason: a caller names an id, never a cart, so the cart it actually
// belongs to is the only one that line can ever be deleted from.
func (s *Store) DeleteCartItemForCart(ctx context.Context, cartID int64, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return models.ErrNotFound
	}
	n, err := s.q.DeleteCartItemForCart(ctx, DeleteCartItemForCartParams{ID: pk, CartID: cartID})
	if err != nil {
		return fmt.Errorf("store: delete cart item %s for cart %d: %w", id, cartID, err)
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

// CartContents is the cart's line items, each joined with the product it
// names, in the order they were added (oldest first).
func (s *Store) CartContents(ctx context.Context, cartID int64) ([]models.CartLine, error) {
	rows, err := s.q.ListCartItemsForCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("store: list cart items for cart %d: %w", cartID, err)
	}
	out := make([]models.CartLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.CartLine{
			ID:             r.ID,
			ProductID:      r.ProductID,
			ProductName:    r.Name,
			ImageUrl:       r.ImageUrl,
			UnitPriceCents: r.PriceCents,
			Quantity:       r.Quantity,
		})
	}
	return out, nil
}
