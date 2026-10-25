package seeds

import (
	"context"

	"ecommerce/app/domain"
	"ecommerce/app/models"
)

// products is the catalog SeedProduct inserts. Edit this list rather than
// the loop below when the catalog changes. Images live in
// public/images/products, embedded into the binary like everything else
// under public/ (see public/public.go), so they need no network to load.
var products = []domain.Product{
	{Name: "Bogie Logo Sticker", Description: "Die-cut vinyl sticker, 3 inches wide. Waterproof and laptop-safe.", PriceCents: 300, Stock: 500, ImageUrl: "/images/products/bogie-logo-sticker.webp"},
	{Name: "Bogie Sticker Pack", Description: "Six vinyl stickers: the logo, the mine cart, and four more from the Bogie track.", PriceCents: 800, Stock: 300, ImageUrl: "/images/products/bogie-sticker-pack.webp"},
	{Name: "Bogie Ceramic Mug", Description: "11 oz white ceramic mug with the Bogie logo. Dishwasher and microwave safe.", PriceCents: 1400, Stock: 150, ImageUrl: "/images/products/bogie-ceramic-mug.webp"},
	{Name: "Bogie Desk Mat", Description: "900x400mm stitched-edge desk mat printed with the Bogie track map.", PriceCents: 2500, Stock: 100, ImageUrl: "/images/products/bogie-desk-mat.webp"},
	{Name: "Bogie Keycap", Description: "Artisan resin keycap with the Bogie mine cart. Fits Cherry MX-style switches.", PriceCents: 1800, Stock: 80, ImageUrl: "/images/products/bogie-keycap.webp"},
	{Name: "Bogie Poster", Description: "18x24 inch print on matte paper: the route from zero to production.", PriceCents: 1500, Stock: 120, ImageUrl: "/images/products/bogie-poster.webp"},
	{Name: "Bogie Notebook", Description: "A5 dot-grid notebook, 120 pages, lay-flat binding, Bogie logo on the cover.", PriceCents: 1200, Stock: 200, ImageUrl: "/images/products/bogie-notebook.webp"},
}

// SeedProduct seeds product. db seed can run this any number of times:
// FindOrCreateProductByName leaves anything already in the catalog alone, so
// re-running only inserts what is missing.
func SeedProduct(ctx context.Context, s *models.Store) error {
	for _, p := range products {
		if _, err := s.FindOrCreateProductByName(ctx, p); err != nil {
			return err
		}
	}
	return nil
}
