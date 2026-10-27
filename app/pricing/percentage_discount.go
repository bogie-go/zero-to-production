package pricing

import (
	"encoding/json"
	"fmt"

	"ecommerce/app/models"
)

// PercentageDiscountRule takes a percentage off the price so far — "20%
// off everything".
type PercentageDiscountRule struct {
	Percentage float64 `json:"percentage"`
}

func init() {
	Register("percentage_discount", func(properties json.RawMessage) (Rule, error) {
		var r PercentageDiscountRule
		if err := json.Unmarshal(properties, &r); err != nil {
			return nil, fmt.Errorf("%w: %s", models.ErrInvalid, err)
		}
		return r, r.validate()
	})
}

func (r PercentageDiscountRule) validate() error {
	if r.Percentage <= 0 || r.Percentage > 100 {
		return fmt.Errorf("%w: percentage must be between 0 and 100", models.ErrInvalid)
	}
	return nil
}

// Apply takes the percentage off whatever CurrentCents already is, not the
// original price — so stacking this after another discount compounds, and
// order matters (20% off then $10 off is not the same as the reverse).
func (r PercentageDiscountRule) Apply(ctx *PriceContext) (map[string]any, error) {
	discount := int64(float64(ctx.CurrentCents) * r.Percentage / 100)
	ctx.CurrentCents -= discount
	return map[string]any{"percentage": r.Percentage, "discount_cents": discount}, nil
}

func init() {
	RegisterSpec(Spec{
		Kind: "percentage_discount", Label: "Percentage off",
		Help:   "Takes a percentage off the price so far, so it compounds with rules that run before it.",
		Fields: []Field{{Name: "percentage", Label: "Percentage off", Type: "percent", Help: "Between 0 and 100."}},
	})
}

// Summary says "20% off".
func (r PercentageDiscountRule) Summary() string { return fmt.Sprintf("%g%% off", r.Percentage) }
