package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// ListCategoriesWithCounts is every category, by name, with how many active
// products it holds (ProductCount), which the shop uses to show only the
// categories worth clicking.
func (s *Store) ListCategoriesWithCounts(ctx context.Context) ([]domain.Category, error) {
	rows, err := s.q.ListCategoriesWithCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("models: list categories: %w", err)
	}
	out := make([]domain.Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Category{
			ID: r.ID, Name: r.Name, ProductCount: r.ProductCount, CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
		})
	}
	return out, nil
}

// GetCategoryByName finds one category by its name.
func (s *Store) GetCategoryByName(ctx context.Context, name string) (domain.Category, error) {
	row, err := s.q.GetCategoryByName(ctx, name)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Category{}, domain.ErrNotFound
	case err != nil:
		return domain.Category{}, fmt.Errorf("models: get category by name %q: %w", name, err)
	}
	return categoryFromRow(row), nil
}

// FindOrCreateCategoryByName returns the category named c.Name, creating it if
// none exists yet: the same find_or_create_by idiom the other seeds use.
func (s *Store) FindOrCreateCategoryByName(ctx context.Context, c domain.Category) (domain.Category, error) {
	existing, err := s.GetCategoryByName(ctx, c.Name)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, domain.ErrNotFound):
		return s.CreateCategory(ctx, c)
	default:
		return domain.Category{}, err
	}
}

// orEmpty reads a nullable text column (a LEFT JOIN's right side, absent when
// there is no match) as the empty string, which is how the domain says "none".
func orEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
