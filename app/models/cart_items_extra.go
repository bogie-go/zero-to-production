package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// AddCartItem adds quantity of a product to a cart, incrementing an
// existing line for that product instead of creating a second one — the
// way adding the same thing to a Rails cart twice usually works. quantity
// must be positive and the line's total, new or incremented, must not
// exceed the product's current stock.
func (s *Store) AddCartItem(ctx context.Context, cartID, productID int64, quantity int32) (domain.CartItem, error) {
	if quantity <= 0 {
		return domain.CartItem{}, fmt.Errorf("%w: quantity must be positive", domain.ErrInvalid)
	}
	product, err := s.q.GetProduct(ctx, productID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.CartItem{}, fmt.Errorf("%w: no such product", domain.ErrInvalid)
	case err != nil:
		return domain.CartItem{}, fmt.Errorf("models: get product %d: %w", productID, err)
	}

	existing, err := s.q.GetCartItemByCartAndProduct(ctx, GetCartItemByCartAndProductParams{CartID: cartID, ProductID: productID})
	switch {
	case err == nil:
		quantity += existing.Quantity
		if quantity > product.Stock {
			return domain.CartItem{}, fmt.Errorf("%w: only %d in stock", domain.ErrInvalid, product.Stock)
		}
		return s.UpdateCartItem(ctx, domain.CartItem{ID: existing.ID, CartID: cartID, ProductID: productID, Quantity: quantity})
	case errors.Is(err, pgx.ErrNoRows):
		if quantity > product.Stock {
			return domain.CartItem{}, fmt.Errorf("%w: only %d in stock", domain.ErrInvalid, product.Stock)
		}
		return s.CreateCartItem(ctx, domain.CartItem{CartID: cartID, ProductID: productID, Quantity: quantity})
	default:
		return domain.CartItem{}, fmt.Errorf("models: get cart item: %w", err)
	}
}

// UpdateCartItemQuantity sets a line's quantity directly, validating it
// against the product's current stock the same way AddCartItem does. id is
// a URL-sourced string, like every other store method that takes one; an
// id that is not a valid id is simply not found.
func (s *Store) UpdateCartItemQuantity(ctx context.Context, id string, quantity int32) (domain.CartItem, error) {
	if quantity <= 0 {
		return domain.CartItem{}, fmt.Errorf("%w: quantity must be positive", domain.ErrInvalid)
	}
	item, err := s.GetCartItem(ctx, id)
	if err != nil {
		return domain.CartItem{}, err
	}
	product, err := s.q.GetProduct(ctx, item.ProductID)
	if err != nil {
		return domain.CartItem{}, fmt.Errorf("models: get product %d: %w", item.ProductID, err)
	}
	if quantity > product.Stock {
		return domain.CartItem{}, fmt.Errorf("%w: only %d in stock", domain.ErrInvalid, product.Stock)
	}
	item.Quantity = quantity
	return s.UpdateCartItem(ctx, item)
}

// CartContents is the cart's line items, each joined with the product it
// names, newest first in the order they were added.
func (s *Store) CartContents(ctx context.Context, cartID int64) ([]domain.CartLine, error) {
	rows, err := s.q.ListCartItemsForCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("models: list cart items for cart %d: %w", cartID, err)
	}
	out := make([]domain.CartLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.CartLine{
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
