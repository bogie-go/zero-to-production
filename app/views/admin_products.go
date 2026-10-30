package views

import (
	"time"
)

// AdminProductResponse is one product on the wire, to staff.
type AdminProductResponse struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	PriceCents  int32      `json:"price_cents"`
	Stock       int32      `json:"stock"`
	ImageUrl    string     `json:"image_url"`
	ArchivedAt  *time.Time `json:"archived_at"`
	// CategoryID is null for an uncategorized product; CategoryName is filled
	// where the response joined the category and empty otherwise.
	CategoryID   *int64    `json:"category_id"`
	CategoryName string    `json:"category_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AdminProductsResponse is a page of a listing.
type AdminProductsResponse struct {
	Products   []AdminProductResponse `json:"products"`
	Pagination Pagination             `json:"pagination"`
}

// AdminProductRequest is the body of POST and PATCH /admin/products. A PATCH
// sends every field, not a partial update — there is no merge-with-existing
// here, the same as every other write in this app.
type AdminProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int32  `json:"price_cents"`
	Stock       int32  `json:"stock"`
	ImageUrl    string `json:"image_url"`
	CategoryID  *int64 `json:"category_id"`
}
