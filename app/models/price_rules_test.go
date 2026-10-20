package models

import (
	"context"
	"errors"
	"testing"

	"ecommerce/app/domain"
)

// Whatever the id looks like, a price_rule that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownPriceRulesAreNotFound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetPriceRule(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetPriceRule(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeletePriceRule(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeletePriceRule(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	if _, err := store.UpdatePriceRule(ctx, domain.PriceRule{ID: 999999999}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdatePriceRule(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

// TODO: a round trip. Create a price_rule with real values, Get it, List it,
// Update it, Delete it; see posts_test.go for the shape.
