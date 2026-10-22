package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

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
