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

// Products, as the models see them. Each method converts models.Product to and from
// the row type sqlc generated, and maps pgx's "no rows" to models.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in products_extra.go, not here.

// CreateProduct validates a product, inserts it, and returns it with its id and
// timestamps. Validation runs on save, so every caller
// gets it and none can forget it.
func (s *Store) CreateProduct(ctx context.Context, p models.Product) (models.Product, error) {
	if err := p.Validate(); err != nil {
		return models.Product{}, err
	}
	row, err := s.q.CreateProduct(ctx, sqlc.CreateProductParams{Name: p.Name, Description: p.Description, PriceCents: p.PriceCents, Stock: p.Stock, ImageUrl: p.ImageUrl})
	if err != nil {
		return models.Product{}, fmt.Errorf("store: create product: %w", err)
	}
	return productFromRow(row), nil
}

// GetProduct finds one product. An id that is not a valid id is simply not found.
func (s *Store) GetProduct(ctx context.Context, id string) (models.Product, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.Product{}, models.ErrNotFound
	}
	row, err := s.q.GetProduct(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Product{}, models.ErrNotFound
	case err != nil:
		return models.Product{}, fmt.Errorf("store: get product %s: %w", id, err)
	}
	return productFromRow(row), nil
}

// ListProducts returns the newest products first, at most limit of them.
func (s *Store) ListProducts(ctx context.Context, limit int32) ([]models.Product, error) {
	rows, err := s.q.ListProducts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list products: %w", err)
	}
	out := make([]models.Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, productFromRow(r))
	}
	return out, nil
}

// UpdateProduct validates a product, writes its columns, and returns the stored row.
func (s *Store) UpdateProduct(ctx context.Context, p models.Product) (models.Product, error) {
	if err := p.Validate(); err != nil {
		return models.Product{}, err
	}
	pk := p.ID
	row, err := s.q.UpdateProduct(ctx, sqlc.UpdateProductParams{ID: pk, Name: p.Name, Description: p.Description, PriceCents: p.PriceCents, Stock: p.Stock, ImageUrl: p.ImageUrl})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.Product{}, models.ErrNotFound
	case err != nil:
		return models.Product{}, fmt.Errorf("store: update product %v: %w", p.ID, err)
	}
	return productFromRow(row), nil
}

// DeleteProduct removes a product, and says so if there was none.
func (s *Store) DeleteProduct(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return models.ErrNotFound
	}
	n, err := s.q.DeleteProduct(ctx, pk)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("%w: product is still in use", models.ErrInvalid)
		}
		return fmt.Errorf("store: delete product %s: %w", id, err)
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func productFromRow(r sqlc.Product) models.Product {
	return models.Product{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		PriceCents:  r.PriceCents,
		Stock:       r.Stock,
		ImageUrl:    r.ImageUrl,
		ArchivedAt:  timestamptzToTimePtr(r.ArchivedAt),
		CreatedAt:   r.CreatedAt.Time,
		UpdatedAt:   r.UpdatedAt.Time,
	}
}
