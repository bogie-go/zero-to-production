package models

// CartLine is one product in a cart, with the product details needed to
// display it: a join of cart_items and products, not a table of its own,
// so there is no CartLine model and no Validate — it is read-only.
type CartLine struct {
	ID             int64
	ProductID      int64
	ProductName    string
	ImageUrl       string
	UnitPriceCents int32
	Quantity       int32
}
