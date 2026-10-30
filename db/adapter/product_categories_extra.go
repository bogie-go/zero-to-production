package adapter

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ecommerce/app/models"
	"ecommerce/db/sqlc"
)

// SetProductCategories makes a product's categories exactly categoryIDs, in
// one transaction: if any category does not exist, nothing changes. A
// category named twice is saved once.
func (s *Store) SetProductCategories(ctx context.Context, productID string, categoryIDs []int64) error {
	pk, err := parseID(productID)
	if err != nil {
		return models.ErrNotFound
	}
	ids := slices.Clone(categoryIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)

	return s.withTx(ctx, func(_ pgx.Tx, q *sqlc.Queries) error {
		_, err := q.GetProductForUpdate(ctx, pk)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("store: set product categories: %w", err)
		}
		if err := q.DeleteProductCategoriesForProduct(ctx, pk); err != nil {
			return fmt.Errorf("store: set product categories: %w", err)
		}
		for _, id := range ids {
			_, err := q.CreateProductCategory(ctx, sqlc.CreateProductCategoryParams{ProductID: pk, CategoryID: id})
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return fmt.Errorf("%w: category %d does not exist", models.ErrInvalid, id)
			}
			if err != nil {
				return fmt.Errorf("store: set product categories: %w", err)
			}
		}
		return nil
	})
}
