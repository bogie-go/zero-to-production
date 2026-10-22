package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
)

// Whatever the id looks like, a category that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownCategoriesAreNotFound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetCategory(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetCategory(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteCategory(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteCategory(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	// A valid-looking category, so Validate (no longer a no-op) passes and the
	// not-found check is what actually gets exercised.
	if _, err := store.UpdateCategory(ctx, domain.Category{ID: 999999999, Name: "Nope"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateCategory(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

func TestCategoryRoundTrip(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	created, err := store.CreateCategory(ctx, domain.Category{Name: "Audio"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.GetCategory(ctx, strconv.FormatInt(created.ID, 10))
	if err != nil || got != created {
		t.Fatalf("GetCategory = %+v, %v; want %+v", got, err, created)
	}
	created.Name = "Sound"
	updated, err := store.UpdateCategory(ctx, created)
	if err != nil || updated.Name != "Sound" {
		t.Fatalf("UpdateCategory = %+v, %v", updated, err)
	}
	if err := store.DeleteCategory(ctx, strconv.FormatInt(created.ID, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetCategory(ctx, strconv.FormatInt(created.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetCategory after delete: err = %v, want ErrNotFound", err)
	}
}

// The name is required and unique, and both are the caller's 422.
func TestCategoryNamesAreRequiredAndUnique(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	if _, err := store.CreateCategory(ctx, domain.Category{Name: "  "}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("blank name: err = %v, want ErrInvalid", err)
	}
	if _, err := store.CreateCategory(ctx, domain.Category{Name: "Audio"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateCategory(ctx, domain.Category{Name: "Audio"}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("duplicate name: err = %v, want ErrInvalid", err)
	}
}
