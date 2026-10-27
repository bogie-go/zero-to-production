package seeds

import (
	"context"
	"errors"
	"fmt"

	"ecommerce/app/models"
	"ecommerce/db/adapter"
)

// categorized is the demo catalog's categories and which seeded products
// belong in each, by name. Edit this list rather than the loop below.
var categorized = []struct {
	name     string
	products []string
}{
	{"Stickers", []string{"Bogie Logo Sticker", "Bogie Sticker Pack"}},
	{"Drinkware", []string{"Bogie Ceramic Mug"}},
	{"Desk", []string{"Bogie Desk Mat", "Bogie Keycap"}},
	{"Paper", []string{"Bogie Poster", "Bogie Notebook"}},
}

// SeedCategories seeds categories, then puts the seeded products in them. db
// seed can run this any number of times: a category that exists is left alone,
// and a product that already has a category (one an admin chose) is never
// moved, so re-running cannot undo anyone's edit.
func SeedCategories(ctx context.Context, s *adapter.Store) error {
	for _, c := range categorized {
		category, err := s.FindOrCreateCategoryByName(ctx, models.Category{Name: c.name})
		if err != nil {
			return err
		}
		for _, name := range c.products {
			product, err := s.GetProductByName(ctx, name)
			if errors.Is(err, models.ErrNotFound) {
				continue // not in this database's catalog; nothing to file
			}
			if err != nil {
				return err
			}
			if product.CategoryID != nil {
				continue
			}
			product.CategoryID = &category.ID
			if _, err := s.UpdateProduct(ctx, product); err != nil {
				return fmt.Errorf("seeds: categorize %q: %w", name, err)
			}
		}
	}
	return nil
}
