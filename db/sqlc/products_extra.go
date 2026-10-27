package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/models"
)

// GetProductByName finds one product by its name.
func (s *Store) GetProductByName(ctx context.Context, name string) (models.Product, error) {
	row, err := s.q.GetProductByName(ctx, name)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Product{}, models.ErrNotFound
	case err != nil:
		return models.Product{}, fmt.Errorf("store: get product by name %q: %w", name, err)
	}
	return productFromRow(row), nil
}

// FindOrCreateProductByName returns the product named p.Name, creating it
// from p if none exists yet. Like Rails's find_or_create_by, this is a plain
// find then create, not atomic: a race between two callers can still create
// two products with the same name.
func (s *Store) FindOrCreateProductByName(ctx context.Context, p models.Product) (models.Product, error) {
	existing, err := s.GetProductByName(ctx, p.Name)
	switch {
	case err == nil:
		return existing, nil
	case errors.Is(err, models.ErrNotFound):
		return s.CreateProduct(ctx, p)
	default:
		return models.Product{}, err
	}
}

// ListActiveProducts is one page of the shop's catalog: products that are not
// archived, newest first, each with its category's name, and only one
// category's when categoryID is given. limit and offset pick the page;
// CountActiveProducts says how many there are in all. ListProducts
// (generated) still returns all of them, archived included.
func (s *Store) ListActiveProducts(ctx context.Context, categoryID *int64, limit, offset int32) ([]models.Product, error) {
	rows, err := s.q.ListActiveProducts(ctx, ListActiveProductsParams{CategoryID: categoryID, RowLimit: limit, RowOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("store: list active products: %w", err)
	}
	out := make([]models.Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.Product{
			ID: r.ID, Name: r.Name, Description: r.Description, PriceCents: r.PriceCents, Stock: r.Stock,
			ImageUrl: r.ImageUrl, ArchivedAt: timestamptzToTimePtr(r.ArchivedAt), CategoryID: r.CategoryID,
			CategoryName: orEmpty(r.CategoryName), CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
		})
	}
	return out, nil
}

// CountActiveProducts is how many products ListActiveProducts would return
// across all its pages, for the same category filter.
func (s *Store) CountActiveProducts(ctx context.Context, categoryID *int64) (int64, error) {
	n, err := s.q.CountActiveProducts(ctx, categoryID)
	if err != nil {
		return 0, fmt.Errorf("store: count active products: %w", err)
	}
	return n, nil
}

// ListProductsPage is one page of the admin's list, newest first:
// every product, or with includeArchived false only those not archived.
func (s *Store) ListProductsPage(ctx context.Context, includeArchived bool, limit, offset int32) ([]models.Product, error) {
	rows, err := s.q.ListProductsPage(ctx, ListProductsPageParams{IncludeArchived: includeArchived, RowLimit: limit, RowOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("store: list products page: %w", err)
	}
	out := make([]models.Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, productFromRow(r))
	}
	return out, nil
}

// CountProducts is how many products ListProductsPage covers in all.
func (s *Store) CountProducts(ctx context.Context, includeArchived bool) (int64, error) {
	n, err := s.q.CountProducts(ctx, includeArchived)
	if err != nil {
		return 0, fmt.Errorf("store: count products: %w", err)
	}
	return n, nil
}

// GetActiveProduct finds one product the shop is still selling, with its
// category's name. An archived product is not found here, exactly as a
// nonexistent one is.
func (s *Store) GetActiveProduct(ctx context.Context, id string) (models.Product, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.Product{}, models.ErrNotFound
	}
	r, err := s.q.GetActiveProduct(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Product{}, models.ErrNotFound
	case err != nil:
		return models.Product{}, fmt.Errorf("store: get active product %s: %w", id, err)
	}
	return models.Product{
		ID: r.ID, Name: r.Name, Description: r.Description, PriceCents: r.PriceCents, Stock: r.Stock,
		ImageUrl: r.ImageUrl, ArchivedAt: timestamptzToTimePtr(r.ArchivedAt), CategoryID: r.CategoryID,
		CategoryName: orEmpty(r.CategoryName), CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
	}, nil
}

// ArchiveProduct retires a product: the soft delete. Archiving one already
// archived changes nothing.
func (s *Store) ArchiveProduct(ctx context.Context, id string) (models.Product, error) {
	return s.setArchived(ctx, id, s.q.ArchiveProduct)
}

// RestoreProduct puts an archived product back on sale.
func (s *Store) RestoreProduct(ctx context.Context, id string) (models.Product, error) {
	return s.setArchived(ctx, id, s.q.RestoreProduct)
}

func (s *Store) setArchived(ctx context.Context, id string, run func(context.Context, int64) (Product, error)) (models.Product, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.Product{}, models.ErrNotFound
	}
	row, err := run(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Product{}, models.ErrNotFound
	case err != nil:
		return models.Product{}, fmt.Errorf("store: archive product %s: %w", id, err)
	}
	return productFromRow(row), nil
}
