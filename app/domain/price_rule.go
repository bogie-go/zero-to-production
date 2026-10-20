package domain

import (
	"encoding/json"
	"time"
)

// PriceRule is a price_rule. Imports nothing third-party: ids are int64s here and
// become bigint at the store boundary.
type PriceRule struct {
	ID         int64
	Name       string
	Kind       string
	Properties json.RawMessage
	Active     bool
	Priority   int32
	StartsAt   *time.Time
	EndsAt     *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Validate is the model's validations, as Rails would put them. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule:
//
//	return fmt.Errorf("%w: title is required", ErrInvalid)
func (p PriceRule) Validate() error {
	return nil
}
