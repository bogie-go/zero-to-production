package models

import (
	"fmt"
	"strings"
	"time"
)

// Category groups products: a product can be in many, through product_categories. Imports nothing
// third-party: ids are int64s here and become bigint at the store boundary.
// ProductCount is how many active products it has, filled only by the query
// that counts them (the shop's category list); it is never written back.
type Category struct {
	ID           int64
	Name         string
	ProductCount int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate is the model's validations. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule.
func (c Category) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalid)
	}
	return nil
}
