package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/models"
)

// ListCategoriesWithCounts is every category, by name, with how many active
// products it holds (ProductCount), which the shop uses to show only the
// categories worth clicking.
func (s *Store) ListCategoriesWithCounts(ctx context.Context) ([]models.Category, error) {
	rows, err := s.q.ListCategoriesWithCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list categories: %w", err)
	}
	out := make([]models.Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.Category{
			ID: r.ID, Name: r.Name, ProductCount: r.ProductCount, CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
		})
	}
	return out, nil
}

// GetCategoryByName finds one category by its name.
func (s *Store) GetCategoryByName(ctx context.Context, name string) (models.Category, error) {
	row, err := s.q.GetCategoryByName(ctx, name)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Category{}, models.ErrNotFound
	case err != nil:
		return models.Category{}, fmt.Errorf("store: get category by name %q: %w", name, err)
	}
	return categoryFromRow(row), nil
}

// FindOrCreateCategoryByName returns the category named c.Name, creating it if
// none exists yet: the same find_or_create_by idiom the other seeds use.
func (s *Store) FindOrCreateCategoryByName(ctx context.Context, c models.Category) (models.Category, error) {
	existing, err := s.GetCategoryByName(ctx, c.Name)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, models.ErrNotFound):
		return s.CreateCategory(ctx, c)
	default:
		return models.Category{}, err
	}
}

// orEmpty reads a nullable text column (a LEFT JOIN's right side, absent when
// there is no match) as the empty string, which is how the models say "none".
func orEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
