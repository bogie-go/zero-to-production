package pricing

import (
	"encoding/json"
	"fmt"

	"ecommerce/app/domain"
)

// FixedDiscountRule subtracts a fixed amount from the price — "$50 off".
type FixedDiscountRule struct {
	AmountCents int64 `json:"amount_cents"`
}

func init() {
	Register("fixed_discount", func(properties json.RawMessage) (Rule, error) {
		var r FixedDiscountRule
		if err := json.Unmarshal(properties, &r); err != nil {
			return nil, fmt.Errorf("%w: %s", domain.ErrInvalid, err)
		}
		return r, r.validate()
	})
}

func (r FixedDiscountRule) validate() error {
	if r.AmountCents < 0 {
		return fmt.Errorf("%w: amount_cents must not be negative", domain.ErrInvalid)
	}
	return nil
}

// Apply never takes the price below zero, the same guard the Rails version
// uses ([context.current_price - discount, 0].max).
func (r FixedDiscountRule) Apply(ctx *PriceContext) (map[string]any, error) {
	discount := r.AmountCents
	if discount > ctx.CurrentCents {
		discount = ctx.CurrentCents
	}
	ctx.CurrentCents -= discount
	return map[string]any{"amount_cents": r.AmountCents, "discount_cents": discount}, nil
}

func init() {
	RegisterSpec(Spec{
		Kind: "fixed_discount", Label: "Fixed amount off",
		Help:   "Takes a fixed amount off the order, never below zero.",
		Fields: []Field{{Name: "amount_cents", Label: "Amount off", Type: "cents"}},
	})
}

// Summary says "$50.00 off".
func (r FixedDiscountRule) Summary() string { return money(r.AmountCents) + " off" }
