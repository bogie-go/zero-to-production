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
