package views

// CategoryResponse is one category on the wire. ProductCount is how many
// active products it holds, so the shop can show only the ones worth opening.
type CategoryResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ProductCount int64  `json:"product_count"`
}

// CategoriesResponse is a listing.
type CategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
}

// CategoryRequest is the body of POST and PATCH /admin/categories.
type CategoryRequest struct {
	Name string `json:"name"`
}
