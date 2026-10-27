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

// Categories, as the models see them. Each method converts models.Category to and from
// the row type sqlc generated, and maps pgx's "no rows" to models.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in categories_extra.go, not here.

// CreateCategory validates a category, inserts it, and returns it with its id and
// timestamps. Validation runs on save, so every caller
// gets it and none can forget it.
func (s *Store) CreateCategory(ctx context.Context, c models.Category) (models.Category, error) {
	if err := c.Validate(); err != nil {
		return models.Category{}, err
	}
	row, err := s.q.CreateCategory(ctx, c.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Category{}, fmt.Errorf("%w: name is already taken", models.ErrInvalid)
		}
		return models.Category{}, fmt.Errorf("store: create category: %w", err)
	}
	return categoryFromRow(row), nil
}

// GetCategory finds one category. An id that is not a valid id is simply not found.
func (s *Store) GetCategory(ctx context.Context, id string) (models.Category, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.Category{}, models.ErrNotFound
	}
	row, err := s.q.GetCategory(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Category{}, models.ErrNotFound
	case err != nil:
		return models.Category{}, fmt.Errorf("store: get category %s: %w", id, err)
	}
	return categoryFromRow(row), nil
}

// ListCategories returns the newest categories first, at most limit of them.
func (s *Store) ListCategories(ctx context.Context, limit int32) ([]models.Category, error) {
	rows, err := s.q.ListCategories(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list categories: %w", err)
	}
	out := make([]models.Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, categoryFromRow(r))
	}
	return out, nil
}

// UpdateCategory validates a category, writes its columns, and returns the stored row.
func (s *Store) UpdateCategory(ctx context.Context, c models.Category) (models.Category, error) {
	if err := c.Validate(); err != nil {
		return models.Category{}, err
	}
	pk := c.ID
	row, err := s.q.UpdateCategory(ctx, sqlc.UpdateCategoryParams{ID: pk, Name: c.Name})
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Category{}, models.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return models.Category{}, fmt.Errorf("%w: name is already taken", models.ErrInvalid)
	case err != nil:
		return models.Category{}, fmt.Errorf("store: update category %v: %w", c.ID, err)
	}
	return categoryFromRow(row), nil
}

// DeleteCategory removes a category, and says so if there was none.
func (s *Store) DeleteCategory(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return models.ErrNotFound
	}
	n, err := s.q.DeleteCategory(ctx, pk)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("%w: category is still in use", models.ErrInvalid)
		}
		return fmt.Errorf("store: delete category %s: %w", id, err)
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func categoryFromRow(r sqlc.Category) models.Category {
	return models.Category{
		ID:        r.ID,
		Name:      r.Name,
		CreatedAt: r.CreatedAt.Time,
		UpdatedAt: r.UpdatedAt.Time,
	}
}
