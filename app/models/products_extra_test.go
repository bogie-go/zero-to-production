package models_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
	"ecommerce/test/testdb"
)

// An archived product leaves the shop (list and get) but the admin's own
// listing still has it, and restoring puts it back.
func TestArchiveHidesAProductFromTheShopAndRestoreBringsItBack(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	p := factories.Product(t, store)
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
	if active, _ := store.ListActiveProducts(ctx, nil, 10, 0); len(active) != 0 {
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
	store := testdb.Store(t)
	ctx := context.Background()
	p := factories.Product(t, store)
	archived, _ := store.ArchiveProduct(ctx, strconv.FormatInt(p.ID, 10))

	archived.Stock = 3
	updated, err := store.UpdateProduct(ctx, archived)
	if err != nil || updated.ArchivedAt == nil {
		t.Errorf("UpdateProduct: %+v, %v; want still archived", updated, err)
	}
}

func TestArchiveUnknownProductsAreNotFound(t *testing.T) {
	store := testdb.Store(t)
	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.ArchiveProduct(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("ArchiveProduct(%q): err = %v, want ErrNotFound", id, err)
		}
	}
}

// A product in an order cannot be hard-deleted: the database refuses, and
// the store answers ErrInvalid (a 422 upstream), not a raw error.
func TestDeleteAProductInAnOrderIsRefused(t *testing.T) {
	store := testdb.Store(t)
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
	store := testdb.Store(t)
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

// Pages are slices of one total order: no product twice, none skipped, newest
// first, and a page past the end is empty rather than an error. The count
// follows the same filter, so the pager and the pages agree.
func TestListActiveProductsPages(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	audio := factories.Category(t, store, func(c *domain.Category) { c.Name = "Audio" })
	for _, n := range []string{"A", "B", "C", "D", "E"} {
		factories.Product(t, store, func(p *domain.Product) { p.Name = n; p.CategoryID = &audio.ID })
	}
	factories.Product(t, store, func(p *domain.Product) { p.Name = "Other"; p.CategoryID = nil })
	archived := factories.Product(t, store, func(p *domain.Product) { p.Name = "Gone"; p.CategoryID = nil })
	if _, err := store.ArchiveProduct(ctx, strconv.FormatInt(archived.ID, 10)); err != nil {
		t.Fatal(err)
	}

	var seen []string
	for offset := int32(0); offset < 6; offset += 2 {
		page, err := store.ListActiveProducts(ctx, nil, 2, offset)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range page {
			seen = append(seen, p.Name)
		}
	}
	want := []string{"Other", "E", "D", "C", "B", "A"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Errorf("pages of 2 = %v, want %v (newest first, archived left out)", seen, want)
	}
	if past, err := store.ListActiveProducts(ctx, nil, 2, 100); err != nil || len(past) != 0 {
		t.Errorf("a page past the end = %v, %v; want empty", past, err)
	}

	if n, err := store.CountActiveProducts(ctx, nil); err != nil || n != 6 {
		t.Errorf("count = %d, %v; want 6 (the archived one not counted)", n, err)
	}
	if n, err := store.CountActiveProducts(ctx, &audio.ID); err != nil || n != 5 {
		t.Errorf("count in Audio = %d, %v; want 5", n, err)
	}
}

// The admin's pages can hold the archived or leave them out, and the count
// follows.
func TestListProductsPageIncludesArchivedOnRequest(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	factories.Product(t, store, func(p *domain.Product) { p.Name = "Live"; p.CategoryID = nil })
	gone := factories.Product(t, store, func(p *domain.Product) { p.Name = "Gone"; p.CategoryID = nil })
	if _, err := store.ArchiveProduct(ctx, strconv.FormatInt(gone.ID, 10)); err != nil {
		t.Fatal(err)
	}

	all, err := store.ListProductsPage(ctx, true, 10, 0)
	if err != nil || len(all) != 2 {
		t.Fatalf("with archived = %v, %v; want 2", all, err)
	}
	live, err := store.ListProductsPage(ctx, false, 10, 0)
	if err != nil || len(live) != 1 || live[0].Name != "Live" {
		t.Fatalf("without archived = %v, %v; want only Live", live, err)
	}
	if n, _ := store.CountProducts(ctx, true); n != 2 {
		t.Errorf("count with archived = %d, want 2", n)
	}
	if n, _ := store.CountProducts(ctx, false); n != 1 {
		t.Errorf("count without archived = %d, want 1", n)
	}
	second, err := store.ListProductsPage(ctx, true, 1, 1)
	if err != nil || len(second) != 1 || second[0].Name != "Live" {
		t.Errorf("page 2 of size 1 = %v, %v; want Live", second, err)
	}
}
