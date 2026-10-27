package models

// OrderLine is one product in an order, with the product details needed to
// display it — the same shape CartLine plays for a cart, and for the same
// reason: a join of order_items and products, not a table of its own.
type OrderLine struct {
	ProductID      int64
	ProductName    string
	ImageUrl       string
	UnitPriceCents int32
	Quantity       int32
}
