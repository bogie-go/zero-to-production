package views

// AdminCategoryResponse is one category on the wire, to staff. ProductCount is
// how many active products it holds.
type AdminCategoryResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ProductCount int64  `json:"product_count"`
}

// AdminCategoriesResponse is a listing.
type AdminCategoriesResponse struct {
	Categories []AdminCategoryResponse `json:"categories"`
}

// AdminCategoryRequest is the body of POST and PATCH /admin/categories.
type AdminCategoryRequest struct {
	Name string `json:"name"`
}
