package models_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
	"ecommerce/test/testdb"
)

func TestCreateUserRejectsADuplicateEmail(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	if _, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "another-digest", Role: domain.RoleCustomer}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("CreateUser(duplicate email): err = %v, want ErrInvalid", err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()

	created, err := store.CreateUser(ctx, domain.User{Email: "a@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer})
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

// The hook runs inside the signup's transaction, with the new user, and what
// it writes commits with the user.
func TestRegisterUserRunsTheHookInsideTheTransaction(t *testing.T) {
	store := testdb.Store(t)
	pool := testdb.Pool(t)
	ctx := context.Background()

	var seen domain.User
	store.OnUserRegistered(func(ctx context.Context, tx pgx.Tx, user domain.User) error {
		seen = user
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE id = $1", user.ID).Scan(&n); err != nil || n != 0 {
			t.Errorf("outside the transaction the user is visible before commit: n=%d err=%v", n, err)
		}
		return nil
	})
	user, err := store.RegisterUser(ctx, domain.User{Email: "new@example.com", PasswordDigest: "digest", Role: domain.RoleCustomer})
	if err != nil {
		t.Fatal(err)
	}
	if seen.ID != user.ID || seen.Email != "new@example.com" {
		t.Errorf("hook saw %+v, want the registered user %+v", seen, user)
	}
	if got, err := store.GetUserByEmail(ctx, "new@example.com"); err != nil || got.ID != user.ID {
		t.Errorf("the user was not committed: %v %v", got, err)
	}
}

// A hook that fails (the queue is down) means no account: the signup is
// refused rather than leaving a user who will never get their email.
func TestAFailingOnUserRegisteredRollsBackTheSignup(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	store.OnUserRegistered(func(context.Context, pgx.Tx, domain.User) error { return errors.New("queue is down") })

	if _, err := store.RegisterUser(ctx, domain.User{Email: "new@example.com", PasswordDigest: "d", Role: domain.RoleCustomer}); err == nil {
		t.Fatal("RegisterUser succeeded although the hook failed")
	}
	if _, err := store.GetUserByEmail(ctx, "new@example.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("the user exists after a failed signup: err = %v", err)
	}
}

// Same answers as CreateUser: a duplicate email is ErrInvalid and runs no
// hook; an invalid user is refused before anything is written.
func TestRegisterUserRefusesDuplicatesAndInvalidUsers(t *testing.T) {
	store := testdb.Store(t)
	ctx := context.Background()
	calls := 0
	store.OnUserRegistered(func(context.Context, pgx.Tx, domain.User) error { calls++; return nil })
	good := domain.User{Email: "a@example.com", PasswordDigest: "d", Role: domain.RoleCustomer}

	if _, err := store.RegisterUser(ctx, good); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RegisterUser(ctx, good); !errors.Is(err, domain.ErrInvalid) || !strings.Contains(err.Error(), "already taken") {
		t.Errorf("duplicate: err = %v, want ErrInvalid already taken", err)
	}
	if _, err := store.RegisterUser(ctx, domain.User{Email: "nope", PasswordDigest: "d", Role: domain.RoleCustomer}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("invalid email: err = %v, want ErrInvalid", err)
	}
	if calls != 1 {
		t.Errorf("hook ran %d times, want 1 (only the first signup)", calls)
	}
}
