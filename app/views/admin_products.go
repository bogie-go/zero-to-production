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
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
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
}

// AdminProductDetailResponse is one product with the categories it is in.
type AdminProductDetailResponse struct {
	AdminProductResponse
	Categories []CategoryNameResponse `json:"categories"`
}

// ProductCategoriesResponse is the categories a product is in.
type ProductCategoriesResponse struct {
	Categories []CategoryNameResponse `json:"categories"`
}

// ProductCategoriesRequest is the body of PUT /admin/products/:id/categories:
// the whole set the product should be in afterwards.
type ProductCategoriesRequest struct {
	CategoryIDs []int64 `json:"category_ids"`
}
