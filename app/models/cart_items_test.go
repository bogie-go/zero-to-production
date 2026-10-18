package models

import (
	"context"
	"errors"
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

	if _, err := store.UpdateCartItem(ctx, domain.CartItem{ID: 999999999}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateCartItem(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

// TODO: a round trip. Create a cart_item with real values, Get it, List it,
// Update it, Delete it; see posts_test.go for the shape.
