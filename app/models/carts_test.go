package models

import (
	"context"
	"errors"
	"testing"

	"ecommerce/app/domain"
)

// Whatever the id looks like, a cart that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownCartsAreNotFound(t *testing.T) {
	store := testStore(t)
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

// TODO: a round trip. Create a cart with real values, Get it, List it,
// Update it, Delete it; see posts_test.go for the shape.
