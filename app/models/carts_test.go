package models_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/test/testdb"
)

// Whatever the id looks like, a cart that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownCartsAreNotFound(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetCart(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetCart(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteCart(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteCart(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	if _, err := store.UpdateCart(ctx, domain.Cart{ID: 999999999}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateCart(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

func TestCartRoundTrip(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	created, err := store.CreateCart(ctx, domain.Cart{Token: "visitor-1"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Error("CreateCart did not assign an id")
	}
	got, err := store.GetCart(ctx, strconv.FormatInt(created.ID, 10))
	if err != nil || got != created {
		t.Fatalf("GetCart = %+v, %v; want %+v", got, err, created)
	}
	if all, err := store.ListCarts(ctx, 10); err != nil || len(all) != 1 || all[0].ID != created.ID {
		t.Errorf("ListCarts = %+v, %v; want just the one created above", all, err)
	}
	created.Token = "visitor-2"
	if updated, err := store.UpdateCart(ctx, created); err != nil || updated.Token != "visitor-2" {
		t.Fatalf("UpdateCart = %+v, %v", updated, err)
	}
	if err := store.DeleteCart(ctx, strconv.FormatInt(created.ID, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetCart(ctx, strconv.FormatInt(created.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetCart after delete: err = %v, want ErrNotFound", err)
	}
}
