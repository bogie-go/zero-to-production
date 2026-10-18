package models

import (
	"context"
	"errors"
	"testing"

	"ecommerce/app/domain"
)

// Whatever the id looks like, a product that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownProductsAreNotFound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetProduct(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetProduct(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteProduct(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteProduct(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	if _, err := store.UpdateProduct(ctx, domain.Product{ID: 999999999}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateProduct(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

// TODO: a round trip. Create a product with real values, Get it, List it,
// Update it, Delete it; see posts_test.go for the shape.
