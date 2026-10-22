package views

import (
	"time"
)

// ProductResponse is one product on the wire.
type ProductResponse struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	PriceCents  int32      `json:"price_cents"`
	Stock       int32      `json:"stock"`
	ImageUrl    string     `json:"image_url"`
	ArchivedAt  *time.Time `json:"archived_at"`
	// CategoryID is null for an uncategorized product; CategoryName is filled
	// where the response joined the category (the shop's listing) and empty
	// otherwise.
	CategoryID   *int64    `json:"category_id"`
	CategoryName string    `json:"category_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProductsResponse is a listing.
type ProductsResponse struct {
	Products []ProductResponse `json:"products"`
}

// ProductRequest is the body of POST and PATCH /admin/products. A PATCH
// sends every field, not a partial update — there is no merge-with-existing
// here, the same as every other write in this app.
type ProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int32  `json:"price_cents"`
	Stock       int32  `json:"stock"`
	ImageUrl    string `json:"image_url"`
	CategoryID  *int64 `json:"category_id"`
}
