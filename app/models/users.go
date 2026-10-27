package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ecommerce/app/domain"
	"ecommerce/db/sqlc"
)

// Users, as the domain sees them. Each method converts domain.User to and from
// the row type sqlc generated, and maps pgx's "no rows" to domain.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in users_extra.go, not here.

// CreateUser validates a user, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	if err := u.Validate(); err != nil {
		return domain.User{}, err
	}
	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{Email: u.Email, PasswordDigest: u.PasswordDigest, Role: u.Role})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, fmt.Errorf("%w: email is already taken", domain.ErrInvalid)
		}
		return domain.User{}, fmt.Errorf("models: create user: %w", err)
	}
	return userFromRow(row), nil
}

// GetUser finds one user. An id that is not a valid id is simply not found.
func (s *Store) GetUser(ctx context.Context, id string) (domain.User, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.User{}, domain.ErrNotFound
	}
	row, err := s.q.GetUser(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.User{}, domain.ErrNotFound
	case err != nil:
		return domain.User{}, fmt.Errorf("models: get user %s: %w", id, err)
	}
	return userFromRow(row), nil
}

// ListUsers returns the newest users first, at most limit of them.
func (s *Store) ListUsers(ctx context.Context, limit int32) ([]domain.User, error) {
	rows, err := s.q.ListUsers(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("models: list users: %w", err)
	}
	out := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		out = append(out, userFromRow(r))
	}
	return out, nil
}

// UpdateUser validates a user, writes its columns, and returns the stored row.
func (s *Store) UpdateUser(ctx context.Context, u domain.User) (domain.User, error) {
	if err := u.Validate(); err != nil {
		return domain.User{}, err
	}
	pk := u.ID
	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{ID: pk, Email: u.Email, PasswordDigest: u.PasswordDigest, Role: u.Role})
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.User{}, domain.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return domain.User{}, fmt.Errorf("%w: email is already taken", domain.ErrInvalid)
	case err != nil:
		return domain.User{}, fmt.Errorf("models: update user %v: %w", u.ID, err)
	}
	return userFromRow(row), nil
}

// DeleteUser removes a user, and says so if there was none.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return domain.ErrNotFound
	}
	n, err := s.q.DeleteUser(ctx, pk)
	if err != nil {
		return fmt.Errorf("models: delete user %s: %w", id, err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func userFromRow(r sqlc.User) domain.User {
	return domain.User{
		ID:             r.ID,
		Email:          r.Email,
		PasswordDigest: r.PasswordDigest,
		Role:           r.Role,
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
	}
}
