package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ecommerce/app/domain"
)

// GetUserByEmail finds one user by their email — a login's first step.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.User{}, domain.ErrNotFound
	case err != nil:
		return domain.User{}, fmt.Errorf("models: get user by email %q: %w", email, err)
	}
	return userFromRow(row), nil
}

// OnUserRegistered registers a function RegisterUser calls inside its
// transaction, after the user is written, with that transaction: the way the
// app enqueues the welcome email so the job exists if and only if the user
// does. app/application.go sets it; tests that need no job leave it unset.
func (s *Store) OnUserRegistered(fn func(ctx context.Context, tx pgx.Tx, user domain.User) error) {
	s.onUserRegistered = fn
}

// RegisterUser is CreateUser for a person signing up: the same validation and
// the same duplicate-email answer (ErrInvalid), but in a transaction that also
// runs the OnUserRegistered hook, so what the hook does (the welcome email's
// job) commits with the user or not at all. Seeds and anything that is not a
// person signing up use CreateUser.
func (s *Store) RegisterUser(ctx context.Context, u domain.User) (domain.User, error) {
	if err := u.Validate(); err != nil {
		return domain.User{}, err
	}
	var user domain.User
	err := s.withTx(ctx, func(q *Queries) error {
		row, err := q.CreateUser(ctx, CreateUserParams{Email: u.Email, PasswordDigest: u.PasswordDigest, Role: u.Role})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return fmt.Errorf("%w: email is already taken", domain.ErrInvalid)
			}
			return fmt.Errorf("models: register user: %w", err)
		}
		user = userFromRow(row)
		if s.onUserRegistered != nil {
			tx, ok := txOf(q)
			if !ok {
				return errors.New("models: onUserRegistered needs a transaction")
			}
			if err := s.onUserRegistered(ctx, tx, user); err != nil {
				return fmt.Errorf("models: after user %d: %w", user.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// FindOrCreateUserByEmail returns the user with u.Email, creating it from u
// if none exists yet — the same find_or_create_by idiom
// FindOrCreateProductByName uses, for the same reason: a seed (the one
// staff account, in db/seeds/users.go) must be safe to run more than once.
// It never updates an existing row: a seed re-run cannot silently reset a
// password or demote a role someone changed by hand since.
func (s *Store) FindOrCreateUserByEmail(ctx context.Context, u domain.User) (domain.User, error) {
	existing, err := s.GetUserByEmail(ctx, u.Email)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, domain.ErrNotFound):
		return s.CreateUser(ctx, u)
	default:
		return domain.User{}, err
	}
}
