package domain

import (
	"fmt"
	"time"
)

// OrderItem is a order_item. Imports nothing third-party: ids are int64s here
// and become bigint at the store boundary.
//
// UnitPriceCents is the product's price at the moment it was ordered, copied
// in rather than looked up live — the same reason Order keeps its own
// totals instead of recomputing them from today's prices.
type OrderItem struct {
	ID             int64
	OrderID        int64
	ProductID      int64
	Quantity       int32
	UnitPriceCents int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Validate is the model's validations, as Rails would put them. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule.
func (o OrderItem) Validate() error {
	if o.OrderID == 0 {
		return fmt.Errorf("%w: order is required", ErrInvalid)
	}
	if o.ProductID == 0 {
		return fmt.Errorf("%w: product is required", ErrInvalid)
	}
	if o.Quantity <= 0 {
		return fmt.Errorf("%w: quantity must be positive", ErrInvalid)
	}
	if o.UnitPriceCents < 0 {
		return fmt.Errorf("%w: unit price cannot be negative", ErrInvalid)
	}
	return nil
}
