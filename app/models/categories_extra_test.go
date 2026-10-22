package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
)

func category(t *testing.T, s *Store, name string) domain.Category {
	t.Helper()
	c, err := s.CreateCategory(context.Background(), domain.Category{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func productIn(t *testing.T, s *Store, name string, categoryID *int64) domain.Product {
	t.Helper()
	p, err := s.CreateProduct(context.Background(), domain.Product{Name: name, PriceCents: 100, Stock: 5, ImageUrl: "/x.webp", CategoryID: categoryID})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// The catalog joins the category's name on, filters by category when asked,
// and an uncategorized product still appears (a LEFT JOIN, not an inner one).
func TestListActiveProductsJoinsAndFiltersByCategory(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	audio, peripherals := category(t, store, "Audio"), category(t, store, "Peripherals")
	productIn(t, store, "Headphones", &audio.ID)
	productIn(t, store, "Mouse", &peripherals.ID)
	productIn(t, store, "Mystery Box", nil)

	all, err := store.ListActiveProducts(ctx, nil, 10, 0)
	if err != nil || len(all) != 3 {
		t.Fatalf("all = %v, %v; want 3 including the uncategorized one", all, err)
	}
	names := map[string]string{}
	for _, p := range all {
		names[p.Name] = p.CategoryName
	}
	if names["Headphones"] != "Audio" || names["Mouse"] != "Peripherals" || names["Mystery Box"] != "" {
		t.Errorf("category names = %v", names)
	}

	only, err := store.ListActiveProducts(ctx, &audio.ID, 10, 0)
	if err != nil || len(only) != 1 || only[0].Name != "Headphones" || only[0].CategoryID == nil || *only[0].CategoryID != audio.ID {
		t.Errorf("audio only = %+v, %v", only, err)
	}
	none := int64(999999999)
	if empty, err := store.ListActiveProducts(ctx, &none, 10, 0); err != nil || len(empty) != 0 {
		t.Errorf("a category with nothing = %v, %v; want an empty list, not an error", empty, err)
	}

	one, err := store.GetActiveProduct(ctx, strconv.FormatInt(all[0].ID, 10))
	if err != nil || one.ID != all[0].ID || one.CategoryName != all[0].CategoryName {
		t.Errorf("GetActiveProduct = %+v, %v; want the same join", one, err)
	}
}

// The counts are of ACTIVE products: an archived one does not make a category
// look stocked, and an empty category still appears, with 0.
func TestListCategoriesWithCountsCountsOnlyActiveProducts(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	audio, empty := category(t, store, "Audio"), category(t, store, "Empty")
	productIn(t, store, "A", &audio.ID)
	b := productIn(t, store, "B", &audio.ID)
	if _, err := store.ArchiveProduct(ctx, strconv.FormatInt(b.ID, 10)); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListCategoriesWithCounts(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
	counts := map[string]int64{}
	for _, c := range got {
		counts[c.Name] = c.ProductCount
	}
	if counts["Audio"] != 1 || counts["Empty"] != 0 {
		t.Errorf("counts = %v, want Audio 1 (one is archived) and Empty 0", counts)
	}
	_ = empty
	if got[0].Name != "Audio" {
		t.Errorf("order = %v, want by name", got)
	}
}

// Deleting a category un-categorizes its products rather than refusing or
// taking them along: ON DELETE SET NULL.
func TestDeletingACategoryUncategorizesItsProducts(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	audio := category(t, store, "Audio")
	p := productIn(t, store, "Headphones", &audio.ID)

	if err := store.DeleteCategory(ctx, strconv.FormatInt(audio.ID, 10)); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetProduct(ctx, strconv.FormatInt(p.ID, 10))
	if err != nil || got.CategoryID != nil {
		t.Errorf("product after its category was deleted = %+v, %v; want it kept, with no category", got, err)
	}
}

// A product pointed at a category that is not there is the caller's 422, on
// create and on update, not a raw foreign-key error.
func TestAProductNeedsARealCategory(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	ghost := int64(999999999)

	if _, err := store.CreateProduct(ctx, domain.Product{Name: "X", ImageUrl: "/x", CategoryID: &ghost}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("create with a missing category: err = %v, want ErrInvalid", err)
	}
	p := productIn(t, store, "Y", nil)
	p.CategoryID = &ghost
	if _, err := store.UpdateProduct(ctx, p); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("update to a missing category: err = %v, want ErrInvalid", err)
	}
}
