package views

// AddCartItemRequest is the body of POST /cart_items.
type AddCartItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
}

// UpdateCartItemRequest is the body of PATCH /cart_items/:id.
type UpdateCartItemRequest struct {
	Quantity int32 `json:"quantity"`
}

// CartLineResponse is one line in a cart, product details included so the
// UI needs no second fetch to render it.
type CartLineResponse struct {
	ID             int64  `json:"id"`
	ProductID      int64  `json:"product_id"`
	Name           string `json:"name"`
	ImageUrl       string `json:"image_url"`
	UnitPriceCents int32  `json:"unit_price_cents"`
	Quantity       int32  `json:"quantity"`
	SubtotalCents  int32  `json:"subtotal_cents"`
}

// CartResponse is the cart on the wire: GET /cart, and what every cart_items
// mutation answers with too, so the UI can always just replace its cart
// state with the response body.
type CartResponse struct {
	Items      []CartLineResponse `json:"items"`
	TotalCents int32              `json:"total_cents"`
}
