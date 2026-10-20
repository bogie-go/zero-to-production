package models

import (
	"context"
	"errors"
	"testing"

	"ecommerce/app/domain"
)

func TestCreateUserRejectsADuplicateEmail(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	if _, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "digest"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "another-digest"}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("CreateUser(duplicate email): err = %v, want ErrInvalid", err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	created, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "digest"})
	if err != nil {
		t.Fatal(err)
	}

	found, err := store.GetUserByEmail(ctx, "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != created.ID {
		t.Errorf("GetUserByEmail: id = %d, want %d", found.ID, created.ID)
	}

	if _, err := store.GetUserByEmail(ctx, "nobody@example.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetUserByEmail(unknown email): err = %v, want ErrNotFound", err)
	}
}
