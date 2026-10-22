package domain

import (
	"fmt"
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
// rule. A line holds at least one: this is the invariant for every writer.
// The store's AddCartItem also refuses a quantity to ADD that is not positive,
// before it is added to a quantity already there, which is a different thing
// (a bad delta, not a bad line).
func (c CartItem) Validate() error {
	if c.Quantity <= 0 {
		return fmt.Errorf("%w: quantity must be positive", ErrInvalid)
	}
	return nil
}
