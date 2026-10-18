package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// GetProductByName finds one product by its name.
func (s *Store) GetProductByName(ctx context.Context, name string) (domain.Product, error) {
	row, err := s.q.GetProductByName(ctx, name)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Product{}, domain.ErrNotFound
	case err != nil:
		return domain.Product{}, fmt.Errorf("models: get product by name %q: %w", name, err)
	}
	return productFromRow(row), nil
}

// FindOrCreateProductByName returns the product named p.Name, creating it
// from p if none exists yet. Like Rails's find_or_create_by, this is a plain
// find then create, not atomic: a race between two callers can still create
// two products with the same name.
func (s *Store) FindOrCreateProductByName(ctx context.Context, p domain.Product) (domain.Product, error) {
	existing, err := s.GetProductByName(ctx, p.Name)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, domain.ErrNotFound):
		return s.CreateProduct(ctx, p)
	default:
		return domain.Product{}, err
	}
}
