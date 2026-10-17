package seeds

import (
	"context"

	"ecommerce/app/domain"
	"ecommerce/app/models"
)

// products is the catalog SeedProduct inserts. Edit this list rather than
// the loop below when the catalog changes.
var products = []domain.Product{
	{Name: "Wireless Mouse", Description: "A 2.4GHz wireless mouse with a 6-month battery life.", PriceCents: 1999, Stock: 120, ImageUrl: "https://picsum.photos/seed/wireless-mouse/600/400"},
	{Name: "Mechanical Keyboard", Description: "Hot-swappable mechanical keyboard with brown switches.", PriceCents: 7999, Stock: 60, ImageUrl: "https://picsum.photos/seed/mechanical-keyboard/600/400"},
	{Name: "USB-C Hub", Description: "7-in-1 USB-C hub with HDMI, SD card, and 100W passthrough.", PriceCents: 3499, Stock: 200, ImageUrl: "https://picsum.photos/seed/usb-c-hub/600/400"},
	{Name: "27-inch Monitor", Description: "27-inch 1440p IPS monitor with a 144Hz refresh rate.", PriceCents: 29999, Stock: 25, ImageUrl: "https://picsum.photos/seed/27-inch-monitor/600/400"},
	{Name: "Noise-Cancelling Headphones", Description: "Over-ear headphones with active noise cancellation and 30-hour battery life.", PriceCents: 19999, Stock: 80, ImageUrl: "https://picsum.photos/seed/noise-cancelling-headphones/600/400"},
	{Name: "Webcam 1080p", Description: "1080p webcam with autofocus and a built-in privacy shutter.", PriceCents: 4999, Stock: 150, ImageUrl: "https://picsum.photos/seed/webcam-1080p/600/400"},
	{Name: "Laptop Stand", Description: "Adjustable aluminum laptop stand, folds flat for travel.", PriceCents: 2999, Stock: 300, ImageUrl: "https://picsum.photos/seed/laptop-stand/600/400"},
	{Name: "Portable SSD 1TB", Description: "1TB USB-C portable SSD with read speeds up to 1050MB/s.", PriceCents: 8999, Stock: 90, ImageUrl: "https://picsum.photos/seed/portable-ssd-1tb/600/400"},
	{Name: "Desk Mat", Description: "900x400mm stitched-edge desk mat, water-resistant surface.", PriceCents: 1499, Stock: 400, ImageUrl: "https://picsum.photos/seed/desk-mat/600/400"},
	{Name: "Wireless Charger", Description: "15W Qi wireless charging pad with LED status indicator.", PriceCents: 2499, Stock: 250, ImageUrl: "https://picsum.photos/seed/wireless-charger/600/400"},
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
