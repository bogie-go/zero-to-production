package models

import (
	"encoding/json"
	"fmt"
	"strings"
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

// Validate is the model's validations. Wrap every
// failure in ErrInvalid so a controller can answer 422 without knowing the
// rule. Whether Properties suit Kind is not checked here: that is the pricing
// package's (each kind validates its own), which imports this one, so the
// caller that saves a rule runs pricing.Load first.
func (p PriceRule) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if strings.TrimSpace(p.Kind) == "" {
		return fmt.Errorf("%w: kind is required", ErrInvalid)
	}
	if p.StartsAt != nil && p.EndsAt != nil && !p.EndsAt.After(*p.StartsAt) {
		return fmt.Errorf("%w: it must end after it starts", ErrInvalid)
	}
	return nil
}
