package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
)

// Whatever the id looks like, a cart_item that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownCartItemsAreNotFound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetCartItem(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetCartItem(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteCartItem(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteCartItem(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	// A valid-looking line, so Validate (no longer a no-op) passes and the
	// not-found check is what actually gets exercised.
	if _, err := store.UpdateCartItem(ctx, domain.CartItem{ID: 999999999, CartID: 1, ProductID: 1, Quantity: 1}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateCartItem(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

func TestCartItemRoundTrip(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	// A cart item names a cart and a product, so both have to exist.
	cart, err := store.CreateCart(ctx, domain.Cart{Token: "visitor-1"})
	if err != nil {
		t.Fatal(err)
	}
	product, err := store.CreateProduct(ctx, domain.Product{Name: "Widget", PriceCents: 500, Stock: 10, ImageUrl: "/w.webp"})
	if err != nil {
		t.Fatal(err)
	}

	created, err := store.CreateCartItem(ctx, domain.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.GetCartItem(ctx, strconv.FormatInt(created.ID, 10))
	if err != nil || got != created {
		t.Fatalf("GetCartItem = %+v, %v; want %+v", got, err, created)
	}
	if all, err := store.ListCartItems(ctx, 10); err != nil || len(all) != 1 || all[0].ID != created.ID {
		t.Errorf("ListCartItems = %+v, %v; want just the one created above", all, err)
	}
	created.Quantity = 5
	if updated, err := store.UpdateCartItem(ctx, created); err != nil || updated.Quantity != 5 {
		t.Fatalf("UpdateCartItem = %+v, %v", updated, err)
	}
	if err := store.DeleteCartItem(ctx, strconv.FormatInt(created.ID, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetCartItem(ctx, strconv.FormatInt(created.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetCartItem after delete: err = %v, want ErrNotFound", err)
	}
}
