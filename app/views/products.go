package views

import (
	"time"
)

// ProductResponse is one product on the wire.
type ProductResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int32     `json:"price_cents"`
	Stock       int32     `json:"stock"`
	ImageUrl    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProductsResponse is a listing.
type ProductsResponse struct {
	Products []ProductResponse `json:"products"`
}
