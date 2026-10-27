package models

import (
	"time"
)

// Cart is a cart. Imports nothing third-party: ids are int64s here and
// become bigint at the store boundary.
type Cart struct {
	ID        int64
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate is the model's validations. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule:
//
//	return fmt.Errorf("%w: title is required", ErrInvalid)
func (c Cart) Validate() error {
	return nil
}
