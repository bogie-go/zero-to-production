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
