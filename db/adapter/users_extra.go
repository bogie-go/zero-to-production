package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ecommerce/app/models"
	"ecommerce/db/sqlc"
)

// GetUserByEmail finds one user by their email — a login's first step.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.User{}, models.ErrNotFound
	case err != nil:
		return models.User{}, fmt.Errorf("store: get user by email %q: %w", email, err)
	}
	return userFromRow(row), nil
}

// OnUserRegistered registers a function RegisterUser calls inside its
// transaction, after the user is written, with that transaction: the way the
// app enqueues the welcome email so the job exists if and only if the user
// does. app/application.go sets it; tests that need no job leave it unset.
func (s *Store) OnUserRegistered(fn func(ctx context.Context, tx pgx.Tx, user models.User) error) {
	s.onUserRegistered = fn
}

// RegisterUser is CreateUser for a person signing up: the same validation and
// the same duplicate-email answer (ErrInvalid), but in a transaction that also
// runs the OnUserRegistered hook, so what the hook does (the welcome email's
// job) commits with the user or not at all. Seeds and anything that is not a
// person signing up use CreateUser.
func (s *Store) RegisterUser(ctx context.Context, u models.User) (models.User, error) {
	if err := u.Validate(); err != nil {
		return models.User{}, err
	}
	var user models.User
	err := s.withTx(ctx, func(tx pgx.Tx, q *sqlc.Queries) error {
		row, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: u.Email, PasswordDigest: u.PasswordDigest, Role: u.Role})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return fmt.Errorf("%w: email is already taken", models.ErrInvalid)
			}
			return fmt.Errorf("store: register user: %w", err)
		}
		user = userFromRow(row)
		if s.onUserRegistered != nil {
			if err := s.onUserRegistered(ctx, tx, user); err != nil {
				return fmt.Errorf("store: after user %d: %w", user.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

// FindOrCreateUserByEmail returns the user with u.Email, creating it from u
// if none exists yet — the same find_or_create_by idiom
// FindOrCreateProductByName uses, for the same reason: a seed (the one
// staff account, in db/seeds/users.go) must be safe to run more than once.
// It never updates an existing row: a seed re-run cannot silently reset a
// password or demote a role someone changed by hand since.
func (s *Store) FindOrCreateUserByEmail(ctx context.Context, u models.User) (models.User, error) {
	existing, err := s.GetUserByEmail(ctx, u.Email)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, models.ErrNotFound):
		return s.CreateUser(ctx, u)
	default:
		return models.User{}, err
	}
}

// PromoteToStaff makes the account with this email a staff member and
// returns it. It is idempotent, and an unknown email is ErrNotFound. This is
// how a real environment gets its first staff account: the person signs up
// like anyone, then an operator promotes them (ecommerce staff promote), so
// no password is ever typed or stored anywhere but the signup.
func (s *Store) PromoteToStaff(ctx context.Context, email string) (models.User, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return models.User{}, err
	}
	if user.Role == models.RoleStaff {
		return user, nil
	}
	user.Role = models.RoleStaff
	return s.UpdateUser(ctx, user)
}
