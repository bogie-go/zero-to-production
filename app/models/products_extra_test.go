package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
)

func widget(t *testing.T, s *Store) domain.Product {
	t.Helper()
	p, err := s.CreateProduct(context.Background(), domain.Product{
		Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 10, ImageUrl: "/w.webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// An archived product leaves the shop (list and get) but the admin's own
// listing still has it, and restoring puts it back.
func TestArchiveHidesAProductFromTheShopAndRestoreBringsItBack(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	p := widget(t, store)
	id := strconv.FormatInt(p.ID, 10)

	archived, err := store.ArchiveProduct(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if archived.ArchivedAt == nil {
		t.Error("ArchiveProduct left ArchivedAt nil")
	}
	if _, err := store.GetActiveProduct(ctx, id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetActiveProduct(archived): err = %v, want ErrNotFound", err)
	}
	if active, _ := store.ListActiveProducts(ctx, 10); len(active) != 0 {
		t.Errorf("ListActiveProducts = %v, want none", active)
	}
	if all, _ := store.ListProducts(ctx, 10); len(all) != 1 || all[0].ArchivedAt == nil {
		t.Errorf("ListProducts = %v, want the archived product still listed for the admin", all)
	}

	// Archiving twice keeps the first time.
	again, err := store.ArchiveProduct(ctx, id)
	if err != nil || !again.ArchivedAt.Equal(*archived.ArchivedAt) {
		t.Errorf("second ArchiveProduct: %v, %v; want the same archived_at", again.ArchivedAt, err)
	}

	restored, err := store.RestoreProduct(ctx, id)
	if err != nil || restored.ArchivedAt != nil {
		t.Fatalf("RestoreProduct: %+v, %v", restored, err)
	}
	if _, err := store.GetActiveProduct(ctx, id); err != nil {
		t.Errorf("GetActiveProduct(restored): %v", err)
	}
}

// Editing an archived product does not quietly bring it back.
func TestUpdateKeepsAProductArchived(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	p := widget(t, store)
	archived, _ := store.ArchiveProduct(ctx, strconv.FormatInt(p.ID, 10))

	archived.Stock = 3
	updated, err := store.UpdateProduct(ctx, archived)
	if err != nil || updated.ArchivedAt == nil {
		t.Errorf("UpdateProduct: %+v, %v; want still archived", updated, err)
	}
}

func TestArchiveUnknownProductsAreNotFound(t *testing.T) {
	store := testStore(t)
	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.ArchiveProduct(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("ArchiveProduct(%q): err = %v, want ErrNotFound", id, err)
		}
	}
}

// A product in an order cannot be hard-deleted: the database refuses, and
// the store answers ErrInvalid (a 422 upstream), not a raw error.
func TestDeleteAProductInAnOrderIsRefused(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	user, cart, product := checkoutFixtures(t, store, 10)
	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProduct(ctx, strconv.FormatInt(product.ID, 10)); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("DeleteProduct(in an order): err = %v, want ErrInvalid", err)
	}
}

// An archived product cannot be added to a cart, and one archived after it
// was added blocks checkout without touching stock or the cart.
func TestArchivedProductsCannotBeBought(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	user, cart, product := checkoutFixtures(t, store, 10)
	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ArchiveProduct(ctx, strconv.FormatInt(product.ID, 10)); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddCartItem(ctx, cart.ID, product.ID, 1); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("AddCartItem(archived): err = %v, want ErrInvalid", err)
	}
	if _, _, err := store.CreateOrderFromCart(ctx, user.ID, cart.ID); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("CreateOrderFromCart(archived line): err = %v, want ErrInvalid", err)
	}
	got, _ := store.GetProduct(ctx, strconv.FormatInt(product.ID, 10))
	if got.Stock != 10 {
		t.Errorf("stock after the refused checkout = %d, want 10", got.Stock)
	}
	if lines, _ := store.CartContents(ctx, cart.ID); len(lines) != 1 {
		t.Errorf("cart after the refused checkout = %v, want the line still there", lines)
	}
}
