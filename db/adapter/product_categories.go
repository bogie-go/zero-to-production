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

// ProductCategories, as the models see them. Each method converts models.ProductCategory to and from
// the row type sqlc generated, and maps pgx's "no rows" to models.ErrNotFound
// so callers never import pgx to ask a question.
//
// Written by `bogie g model`: overwritten whole on the next run. A
// hand-written addition goes in product_categories_extra.go, not here.

// CreateProductCategory validates a product_category, inserts it, and returns it with its id and
// timestamps. Validation runs on save, so every caller
// gets it and none can forget it.
func (s *Store) CreateProductCategory(ctx context.Context, p models.ProductCategory) (models.ProductCategory, error) {
	if err := p.Validate(); err != nil {
		return models.ProductCategory{}, err
	}
	row, err := s.q.CreateProductCategory(ctx, sqlc.CreateProductCategoryParams{ProductID: p.ProductID, CategoryID: p.CategoryID})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return models.ProductCategory{}, fmt.Errorf("%w: product_id or category_id does not exist", models.ErrInvalid)
		}
		return models.ProductCategory{}, fmt.Errorf("store: create product_category: %w", err)
	}
	return productCategoryFromRow(row), nil
}

// GetProductCategory finds one product_category. An id that is not a valid id is simply not found.
func (s *Store) GetProductCategory(ctx context.Context, id string) (models.ProductCategory, error) {
	pk, err := parseID(id)
	if err != nil {
		return models.ProductCategory{}, models.ErrNotFound
	}
	row, err := s.q.GetProductCategory(ctx, pk)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.ProductCategory{}, models.ErrNotFound
	case err != nil:
		return models.ProductCategory{}, fmt.Errorf("store: get product_category %s: %w", id, err)
	}
	return productCategoryFromRow(row), nil
}

// ListProductCategories returns the newest product_categories first, at most limit of them.
func (s *Store) ListProductCategories(ctx context.Context, limit int32) ([]models.ProductCategory, error) {
	rows, err := s.q.ListProductCategories(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list product_categories: %w", err)
	}
	out := make([]models.ProductCategory, 0, len(rows))
	for _, r := range rows {
		out = append(out, productCategoryFromRow(r))
	}
	return out, nil
}

// UpdateProductCategory validates a product_category, writes its columns, and returns the stored row.
func (s *Store) UpdateProductCategory(ctx context.Context, p models.ProductCategory) (models.ProductCategory, error) {
	if err := p.Validate(); err != nil {
		return models.ProductCategory{}, err
	}
	pk := p.ID
	row, err := s.q.UpdateProductCategory(ctx, sqlc.UpdateProductCategoryParams{ID: pk, ProductID: p.ProductID, CategoryID: p.CategoryID})
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.ProductCategory{}, models.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "23503":
		return models.ProductCategory{}, fmt.Errorf("%w: product_id or category_id does not exist", models.ErrInvalid)
	case err != nil:
		return models.ProductCategory{}, fmt.Errorf("store: update product_category %v: %w", p.ID, err)
	}
	return productCategoryFromRow(row), nil
}

// DeleteProductCategory removes a product_category, and says so if there was none.
func (s *Store) DeleteProductCategory(ctx context.Context, id string) error {
	pk, err := parseID(id)
	if err != nil {
		return models.ErrNotFound
	}
	n, err := s.q.DeleteProductCategory(ctx, pk)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("%w: product_category is still in use", models.ErrInvalid)
		}
		return fmt.Errorf("store: delete product_category %s: %w", id, err)
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func productCategoryFromRow(r sqlc.ProductCategory) models.ProductCategory {
	return models.ProductCategory{
		ID:         r.ID,
		ProductID:  r.ProductID,
		CategoryID: r.CategoryID,
		CreatedAt:  r.CreatedAt.Time,
		UpdatedAt:  r.UpdatedAt.Time,
	}
}
