package models_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ecommerce/app/domain"
	"ecommerce/app/services/password"
	"ecommerce/test/testdb"
)

// Whatever the id looks like, a user that is not there is ErrNotFound, and a
// caller cannot tell a malformed id from a well-formed one that names nothing.
func TestUnknownUsersAreNotFound(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	for _, id := range []string{"nope", "", "999999999"} {
		if _, err := store.GetUser(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetUser(%q): err = %v, want ErrNotFound", id, err)
		}
		if err := store.DeleteUser(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteUser(%q): err = %v, want ErrNotFound", id, err)
		}
	}

	// A valid-looking user, so Validate (unlike every other generated
	// model's, no longer a no-op) passes and the not-found check is what
	// actually gets exercised.
	missing := domain.User{ID: 999999999, Email: "nope@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer}
	if _, err := store.UpdateUser(ctx, missing); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateUser(%d): err = %v, want ErrNotFound", 999999999, err)
	}
}

func TestUserRoundTrip(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	digest, err := password.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: digest, Role: domain.RoleCustomer})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Error("CreateUser did not assign an id")
	}

	got, err := store.GetUser(ctx, strconv.FormatInt(created.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Errorf("GetUser = %+v, want %+v", got, created)
	}

	all, err := store.ListUsers(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != created.ID {
		t.Errorf("ListUsers = %+v, want just the one created above", all)
	}

	created.Email = "b@example.com"
	updated, err := store.UpdateUser(ctx, created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Email != "b@example.com" {
		t.Errorf("UpdateUser: email = %q, want b@example.com", updated.Email)
	}

	if err := store.DeleteUser(ctx, strconv.FormatInt(created.ID, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetUser(ctx, strconv.FormatInt(created.ID, 10)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetUser after delete: err = %v, want ErrNotFound", err)
	}
}
