package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ecommerce/app/domain"
)

// Categories, as the domain sees them. Each method converts domain.Category to and from
// the row type sqlc generated, and maps pgx's "no rows" to domain.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in categories_extra.go, not here.

// CreateCategory validates a category, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	if err := c.Validate(); err != nil {
		return domain.Category{}, err
	}
	row, err := s.q.CreateCategory(ctx, c.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Category{}, fmt.Errorf("%w: name is already taken", domain.ErrInvalid)
		}
		return domain.Category{}, fmt.Errorf("models: create category: %w", err)
	}
	return categoryFromRow(row), nil
}

// GetCategory finds one category. An id that is not a valid id is simply not found.
func (s *Store) GetCategory(ctx context.Context, id string) (domain.Category, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.Category{}, domain.ErrNotFound
	}
	row, err := s.q.GetCategory(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Category{}, domain.ErrNotFound
	case err != nil:
		return domain.Category{}, fmt.Errorf("models: get category %s: %w", id, err)
	}
	return categoryFromRow(row), nil
}

// ListCategories returns the newest categories first, at most limit of them.
func (s *Store) ListCategories(ctx context.Context, limit int32) ([]domain.Category, error) {
	rows, err := s.q.ListCategories(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("models: list categories: %w", err)
	}
	out := make([]domain.Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, categoryFromRow(r))
	}
	return out, nil
}

// UpdateCategory validates a category, writes its columns, and returns the stored row.
func (s *Store) UpdateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	if err := c.Validate(); err != nil {
		return domain.Category{}, err
	}
	pk := c.ID
	row, err := s.q.UpdateCategory(ctx, UpdateCategoryParams{ID: pk, Name: c.Name})
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Category{}, domain.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return domain.Category{}, fmt.Errorf("%w: name is already taken", domain.ErrInvalid)
	case err != nil:
		return domain.Category{}, fmt.Errorf("models: update category %v: %w", c.ID, err)
	}
	return categoryFromRow(row), nil
}

// DeleteCategory removes a category, and says so if there was none.
func (s *Store) DeleteCategory(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return domain.ErrNotFound
	}
	n, err := s.q.DeleteCategory(ctx, pk)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("%w: category is still in use", domain.ErrInvalid)
		}
		return fmt.Errorf("models: delete category %s: %w", id, err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func categoryFromRow(r Category) domain.Category {
	return domain.Category{
		ID:        r.ID,
		Name:      r.Name,
		CreatedAt: r.CreatedAt.Time,
		UpdatedAt: r.UpdatedAt.Time,
	}
}
