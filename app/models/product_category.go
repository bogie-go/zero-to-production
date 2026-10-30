package models

import (
	"time"
)

// ProductCategory is a product_category. Imports nothing third-party: ids are int64s here and
// become bigint at the store boundary.
type ProductCategory struct {
	ID         int64
	ProductID  int64
	CategoryID int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Validate is the model's validations. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule:
//
//	return fmt.Errorf("%w: title is required", ErrInvalid)
func (p ProductCategory) Validate() error {
	return nil
}
