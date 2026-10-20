package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
)

// The cross-cart bug from task C2: UpdateCartItemQuantity and
// DeleteCartItemForCart must refuse a line that doesn't belong to the cart
// id they were given, the same way a caller can never update or delete
// another visitor's line by guessing a bigint id.
func TestUpdateAndDeleteCartItemAreScopedToTheirCart(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	cartA, err := store.FindOrCreateCartByToken(ctx, "visitor-a")
	if err != nil {
		t.Fatal(err)
	}
	cartB, err := store.FindOrCreateCartByToken(ctx, "visitor-b")
	if err != nil {
		t.Fatal(err)
	}
	product, err := store.CreateProduct(ctx, domain.Product{
		Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 100, ImageUrl: "/widget.webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	itemA, err := store.AddCartItem(ctx, cartA.ID, product.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	id := strconv.FormatInt(itemA.ID, 10)

	if _, err := store.UpdateCartItemQuantity(ctx, cartB.ID, id, 5); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateCartItemQuantity(cartB, itemA): err = %v, want ErrNotFound", err)
	}
	if err := store.DeleteCartItemForCart(ctx, cartB.ID, id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("DeleteCartItemForCart(cartB, itemA): err = %v, want ErrNotFound", err)
	}

	// Neither wrong-cart call touched the line: it's still there, unchanged.
	still, err := store.GetCartItem(ctx, id)
	if err != nil {
		t.Fatalf("line should still exist after both wrong-cart calls: %v", err)
	}
	if still.Quantity != 2 {
		t.Errorf("quantity = %d, want unchanged at 2", still.Quantity)
	}

	// The owning cart can do both.
	if _, err := store.UpdateCartItemQuantity(ctx, cartA.ID, id, 5); err != nil {
		t.Errorf("UpdateCartItemQuantity(cartA, itemA): %v", err)
	}
	if err := store.DeleteCartItemForCart(ctx, cartA.ID, id); err != nil {
		t.Errorf("DeleteCartItemForCart(cartA, itemA): %v", err)
	}
	if _, err := store.GetCartItem(ctx, id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("line should be gone after its own cart deletes it: err = %v", err)
	}
}
