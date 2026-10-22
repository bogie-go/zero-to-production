package models

import (
	"context"
	"errors"
	"testing"

	"ecommerce/app/domain"
)

func TestGetCartByTokenDoesNotCreateOne(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	if _, err := store.GetCartByToken(ctx, "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetCartByToken(unknown): err = %v, want ErrNotFound", err)
	}
	if _, err := store.FindOrCreateCartByToken(ctx, "nobody"); err != nil {
		t.Fatal(err)
	}
	// FindOrCreateCartByToken just created one; GetCartByToken should now
	// find it without creating a second.
	found, err := store.GetCartByToken(ctx, "nobody")
	if err != nil {
		t.Fatal(err)
	}
	if found.Token != "nobody" {
		t.Errorf("found cart token = %q, want %q", found.Token, "nobody")
	}
}

func TestDeleteCartAndItemsRemovesBoth(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	cart, err := store.FindOrCreateCartByToken(ctx, "guest-abc")
	if err != nil {
		t.Fatal(err)
	}
	product, err := store.CreateProduct(ctx, domain.Product{
		Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 100, ImageUrl: "/widget.webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 2); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteCartAndItems(ctx, cart.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetCartByToken(ctx, "guest-abc"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetCartByToken after delete: err = %v, want ErrNotFound", err)
	}
	lines, err := store.CartContents(ctx, cart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("CartContents after delete = %v, want none", lines)
	}
}

// Two first requests for the same token at the same moment still end up with
// one cart: the unique index plus the single upsert statement, not luck.
func TestFindOrCreateCartByTokenIsSafeUnderConcurrency(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	const n = 8
	ids := make(chan int64, n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			c, err := store.FindOrCreateCartByToken(ctx, "racing-guest")
			ids <- c.ID
			errs <- err
		}()
	}
	first := <-ids
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			if id := <-ids; id != first {
				t.Errorf("got cart %d and %d for one token", first, id)
			}
		}
	}
	if carts, _ := store.ListCarts(ctx, 100); len(carts) != 1 {
		t.Errorf("%d carts exist, want 1", len(carts))
	}
}

// The index is the guarantee itself: a second cart with a taken token is
// refused by the database, whoever tries to create it.
func TestCartTokensAreUnique(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	if _, err := store.CreateCart(ctx, domain.Cart{Token: "same"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateCart(ctx, domain.Cart{Token: "same"}); err == nil {
		t.Error("a second cart with the same token was created")
	}
}
