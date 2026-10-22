package pricing

import (
	"encoding/json"
	"fmt"

	"ecommerce/app/domain"
)

// MinimumPurchaseRule gives a fixed discount, but only once the price so
// far reaches a minimum — "spend $500, get $75 off".
type MinimumPurchaseRule struct {
	MinimumCents  int64 `json:"minimum_cents"`
	DiscountCents int64 `json:"discount_cents"`
}

func init() {
	Register("minimum_purchase", func(properties json.RawMessage) (Rule, error) {
		var r MinimumPurchaseRule
		if err := json.Unmarshal(properties, &r); err != nil {
			return nil, fmt.Errorf("%w: %s", domain.ErrInvalid, err)
		}
		return r, r.validate()
	})
}

func (r MinimumPurchaseRule) validate() error {
	if r.MinimumCents < 0 || r.DiscountCents < 0 {
		return fmt.Errorf("%w: minimum_cents and discount_cents must not be negative", domain.ErrInvalid)
	}
	return nil
}

// Apply returns a nil detail, applying nothing, when the price so far
// hasn't reached MinimumCents — the threshold is checked against whatever
// earlier rules already did, same as every other rule in the pipeline.
func (r MinimumPurchaseRule) Apply(ctx *PriceContext) (map[string]any, error) {
	if ctx.CurrentCents < r.MinimumCents {
		return nil, nil
	}
	discount := r.DiscountCents
	if discount > ctx.CurrentCents {
		discount = ctx.CurrentCents
	}
	ctx.CurrentCents -= discount
	return map[string]any{"minimum_cents": r.MinimumCents, "discount_cents": discount}, nil
}

func init() {
	RegisterSpec(Spec{
		Kind: "minimum_purchase", Label: "Amount off a big order",
		Help: "A fixed discount, but only once the order reaches a minimum.",
		Fields: []Field{
			{Name: "minimum_cents", Label: "Order must reach", Type: "cents"},
			{Name: "discount_cents", Label: "Amount off", Type: "cents"},
		},
	})
}

// Summary says "$75.00 off orders of $500.00 or more".
func (r MinimumPurchaseRule) Summary() string {
	return fmt.Sprintf("%s off orders of %s or more", money(r.DiscountCents), money(r.MinimumCents))
}
