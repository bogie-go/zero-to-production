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

// AppliedRuleResponse is one price rule that affected this cart's total,
// in the order it ran.
type AppliedRuleResponse struct {
	Name          string `json:"name"`
	DiscountCents int32  `json:"discount_cents"`
}

// CartResponse is the cart on the wire: GET /cart, and what every cart_items
// mutation answers with too, so the UI can always just replace its cart
// state with the response body. SubtotalCents is the lines' total before any
// price rule; TotalCents is what the customer actually pays.
type CartResponse struct {
	Items         []CartLineResponse    `json:"items"`
	SubtotalCents int32                 `json:"subtotal_cents"`
	DiscountCents int32                 `json:"discount_cents"`
	TotalCents    int32                 `json:"total_cents"`
	AppliedRules  []AppliedRuleResponse `json:"applied_rules"`
}
