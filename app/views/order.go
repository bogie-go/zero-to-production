package views

import "time"

// OrderLineResponse is one line in an order, the same shape CartLineResponse
// plays for a cart.
type OrderLineResponse struct {
	ProductID      int64  `json:"product_id"`
	Name           string `json:"name"`
	ImageUrl       string `json:"image_url"`
	UnitPriceCents int32  `json:"unit_price_cents"`
	Quantity       int32  `json:"quantity"`
	SubtotalCents  int32  `json:"subtotal_cents"`
}

// OrderResponse is an order on the wire: what POST /orders answers with,
// and what GET /orders/:id answers with later — the receipt, built from
// what was saved at checkout, never recomputed from today's prices.
type OrderResponse struct {
	ID            int64               `json:"id"`
	Status        string              `json:"status"`
	Items         []OrderLineResponse `json:"items"`
	SubtotalCents int32               `json:"subtotal_cents"`
	DiscountCents int32               `json:"discount_cents"`
	TotalCents    int32               `json:"total_cents"`
}

// OrderSummaryResponse is one row of GET /orders: enough to list and link
// into, without every line item.
type OrderSummaryResponse struct {
	ID         int64     `json:"id"`
	Status     string    `json:"status"`
	TotalCents int32     `json:"total_cents"`
	CreatedAt  time.Time `json:"created_at"`
}
