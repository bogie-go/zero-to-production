package domain

import (
	"time"
)

// Product is a product. Imports nothing third-party: ids are int64s here and
// become bigint at the store boundary.
type Product struct {
	ID          int64
	Name        string
	Description string
	PriceCents  int32
	Stock       int32
	ImageUrl    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate is the model's validations, as Rails would put them. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule:
//
//	return fmt.Errorf("%w: title is required", ErrInvalid)
func (p Product) Validate() error {
	return nil
}
