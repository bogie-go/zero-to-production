package models

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
)

// Whatever the id looks like, a order that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownOrdersAreNotFound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetOrder(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetOrder(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteOrder(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteOrder(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	// A valid-looking order (UserID need not name a real user: zero rows
	// match id 999999999, so the foreign key is never actually checked),
	// so Validate passes and the not-found check is what actually gets
	// exercised.
	missing := domain.Order{ID: 999999999, UserID: 1, Status: "pending"}
	if _, err := store.UpdateOrder(ctx, missing); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateOrder(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

func TestOrderRoundTrip(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	user, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "digest"})
	if err != nil {
		t.Fatal(err)
	}

	created, err := store.CreateOrder(ctx, domain.Order{
		UserID: user.ID, Status: "pending", SubtotalCents: 1000, DiscountCents: 100, TotalCents: 900,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Error("CreateOrder did not assign an id")
	}

	got, err := store.GetOrder(ctx, strconv.FormatInt(created.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Errorf("GetOrder = %+v, want %+v", got, created)
	}

	all, err := store.ListOrders(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != created.ID {
		t.Errorf("ListOrders = %+v, want just the one created above", all)
	}

	created.Status = "paid"
	updated, err := store.UpdateOrder(ctx, created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "paid" {
		t.Errorf("UpdateOrder: status = %q, want paid", updated.Status)
	}

	if err := store.DeleteOrder(ctx, strconv.FormatInt(created.ID, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetOrder(ctx, strconv.FormatInt(created.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetOrder after delete: err = %v, want ErrNotFound", err)
	}
}
