package models_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/test/testdb"
)

// Whatever the id looks like, a product that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownProductsAreNotFound(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetProduct(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetProduct(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteProduct(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteProduct(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	// A valid-looking product, so Validate (no longer a no-op, now that
	// admin can create and update one) passes and the not-found check is
	// what actually gets exercised.
	missing := domain.Product{ID: 999999999, Name: "Nonexistent"}
	if _, err := store.UpdateProduct(ctx, missing); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateProduct(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

func TestProductRoundTrip(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	created, err := store.CreateProduct(ctx, domain.Product{
		Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 10, ImageUrl: "/widget.webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Error("CreateProduct did not assign an id")
	}

	got, err := store.GetProduct(ctx, strconv.FormatInt(created.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Errorf("GetProduct = %+v, want %+v", got, created)
	}

	all, err := store.ListProducts(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != created.ID {
		t.Errorf("ListProducts = %+v, want just the one created above", all)
	}

	created.Stock = 7
	updated, err := store.UpdateProduct(ctx, created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Stock != 7 {
		t.Errorf("UpdateProduct: stock = %d, want 7", updated.Stock)
	}

	if err := store.DeleteProduct(ctx, strconv.FormatInt(created.ID, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetProduct(ctx, strconv.FormatInt(created.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetProduct after delete: err = %v, want ErrNotFound", err)
	}
}
