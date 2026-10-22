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

// ListActiveProducts is the shop's catalog: every product that is not
// archived, newest first. ListProducts (generated) still returns all of
// them, archived included, for the admin.
func (s *Store) ListActiveProducts(ctx context.Context, limit int32) ([]domain.Product, error) {
	rows, err := s.q.ListActiveProducts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("models: list active products: %w", err)
	}
	out := make([]domain.Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, productFromRow(r))
	}
	return out, nil
}

// GetActiveProduct finds one product the shop is still selling. An archived
// product is not found here, exactly as a nonexistent one is.
func (s *Store) GetActiveProduct(ctx context.Context, id string) (domain.Product, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.Product{}, domain.ErrNotFound
	}
	row, err := s.q.GetActiveProduct(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Product{}, domain.ErrNotFound
	case err != nil:
		return domain.Product{}, fmt.Errorf("models: get active product %s: %w", id, err)
	}
	return productFromRow(row), nil
}

// ArchiveProduct retires a product: the soft delete. Archiving one already
// archived changes nothing.
func (s *Store) ArchiveProduct(ctx context.Context, id string) (domain.Product, error) {
	return s.setArchived(ctx, id, s.q.ArchiveProduct)
}

// RestoreProduct puts an archived product back on sale.
func (s *Store) RestoreProduct(ctx context.Context, id string) (domain.Product, error) {
	return s.setArchived(ctx, id, s.q.RestoreProduct)
}

func (s *Store) setArchived(ctx context.Context, id string, run func(context.Context, int64) (Product, error)) (domain.Product, error) {
	pk, err := parseID(id)
	if err != nil {
		return domain.Product{}, domain.ErrNotFound
	}
	row, err := run(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Product{}, domain.ErrNotFound
	case err != nil:
		return domain.Product{}, fmt.Errorf("models: archive product %s: %w", id, err)
	}
	return productFromRow(row), nil
}
