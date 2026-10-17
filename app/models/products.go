package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ecommerce/app/domain"
)

// Products, as the domain sees them. Each method converts domain.Product to and from
// the row type sqlc generated, and maps pgx's "no rows" to domain.ErrNotFound
// so callers never import pgx to ask a question.

// CreateProduct validates a product, inserts it, and returns it with its id and
// timestamps. Validation runs on save, as Rails's does, so every caller
// gets it and none can forget it.
func (s *Store) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	if err := p.Validate(); err != nil {
		return domain.Product{}, err
	}
	row, err := s.q.CreateProduct(ctx, CreateProductParams{Name: p.Name, Description: p.Description, PriceCents: p.PriceCents, Stock: p.Stock, ImageUrl: p.ImageUrl})
	if err != nil {
		return domain.Product{}, fmt.Errorf("models: create product: %w", err)
	}
	return productFromRow(row), nil
}

// GetProduct finds one product. An id that is not a uuid is simply not found.
func (s *Store) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return domain.Product{}, domain.ErrNotFound
	}
	row, err := s.q.GetProduct(ctx, uid)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Product{}, domain.ErrNotFound
	case err != nil:
		return domain.Product{}, fmt.Errorf("models: get product %s: %w", id, err)
	}
	return productFromRow(row), nil
}

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

// ListProducts returns the newest products first, at most limit of them.
func (s *Store) ListProducts(ctx context.Context, limit int32) ([]domain.Product, error) {
	rows, err := s.q.ListProducts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("models: list products: %w", err)
	}
	out := make([]domain.Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, productFromRow(r))
	}
	return out, nil
}

// UpdateProduct validates a product, writes its columns, and returns the stored row.
func (s *Store) UpdateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	if err := p.Validate(); err != nil {
		return domain.Product{}, err
	}
	uid, err := parseUUID(p.ID)
	if err != nil {
		return domain.Product{}, domain.ErrNotFound
	}
	row, err := s.q.UpdateProduct(ctx, UpdateProductParams{ID: uid, Name: p.Name, Description: p.Description, PriceCents: p.PriceCents, Stock: p.Stock, ImageUrl: p.ImageUrl})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Product{}, domain.ErrNotFound
	case err != nil:
		return domain.Product{}, fmt.Errorf("models: update product %s: %w", p.ID, err)
	}
	return productFromRow(row), nil
}

// DeleteProduct removes a product, and says so if there was none.
func (s *Store) DeleteProduct(ctx context.Context, id string) error {
	uid, err := parseUUID(id)
	if err != nil {
		return domain.ErrNotFound
	}
	n, err := s.q.DeleteProduct(ctx, uid)
	if err != nil {
		return fmt.Errorf("models: delete product %s: %w", id, err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func productFromRow(r Product) domain.Product {
	return domain.Product{
		ID:          uuidString(r.ID),
		Name:        r.Name,
		Description: r.Description,
		PriceCents:  r.PriceCents,
		Stock:       r.Stock,
		ImageUrl:    r.ImageUrl,
		CreatedAt:   r.CreatedAt.Time,
		UpdatedAt:   r.UpdatedAt.Time,
	}
}
