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
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CategoryNameResponse is a category as a product's page names it.
type CategoryNameResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ProductDetailResponse is one product, with the categories it is in. The
// embedded ProductResponse's fields sit beside categories in the JSON.
type ProductDetailResponse struct {
	ProductResponse
	Categories []CategoryNameResponse `json:"categories"`
}

// ProductsResponse is a page of a listing.
type ProductsResponse struct {
	Products   []ProductResponse `json:"products"`
	Pagination Pagination        `json:"pagination"`
}
