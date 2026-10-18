package domain

import (
	"time"
)

// CartItem is a cart_item. Imports nothing third-party: ids are int64s here and
// become bigint at the store boundary.
type CartItem struct {
	ID        int64
	CartID    int64
	ProductID int64
	Quantity  int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate is the model's validations, as Rails would put them. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule:
//
//	return fmt.Errorf("%w: title is required", ErrInvalid)
func (c CartItem) Validate() error {
	return nil
}
