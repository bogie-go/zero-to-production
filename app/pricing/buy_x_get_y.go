package pricing

import (
	"encoding/json"
	"fmt"

	"ecommerce/app/domain"
)

// BuyXGetYRule counts free items once a quantity threshold is met — "buy 2
// get 1 free". It reads ctx.Metadata["quantity"], the count of whatever is
// being bought; a caller that doesn't set it simply never meets the
// threshold.
//
// Ported as-is from the source article: it records how many items are free
// in ctx.Metadata["free_items"], but — same as the article — does not
// itself discount CurrentCents. Doing that correctly needs the specific
// product's unit price, which a single cart-wide price doesn't carry; the
// real fix is applying this rule per cart line, where the unit price is
// known, not here.
type BuyXGetYRule struct {
	RequiredQuantity int32 `json:"required_quantity"`
	FreeQuantity     int32 `json:"free_quantity"`
}

func init() {
	Register("buy_x_get_y", func(properties json.RawMessage) (Rule, error) {
		var r BuyXGetYRule
		if err := json.Unmarshal(properties, &r); err != nil {
			return nil, fmt.Errorf("%w: %s", domain.ErrInvalid, err)
		}
		return r, r.validate()
	})
}

func (r BuyXGetYRule) validate() error {
	if r.RequiredQuantity <= 0 || r.FreeQuantity <= 0 {
		return fmt.Errorf("%w: required_quantity and free_quantity must be positive", domain.ErrInvalid)
	}
	return nil
}

func (r BuyXGetYRule) Apply(ctx *PriceContext) (map[string]any, error) {
	quantity, _ := ctx.Metadata["quantity"].(int)
	if quantity < int(r.RequiredQuantity) {
		return nil, nil
	}
	sets := quantity / int(r.RequiredQuantity)
	freeItems := sets * int(r.FreeQuantity)
	already, _ := ctx.Metadata["free_items"].(int)
	ctx.Metadata["free_items"] = already + freeItems
	return map[string]any{
		"required_quantity": r.RequiredQuantity,
		"free_quantity":     r.FreeQuantity,
		"free_items":        freeItems,
	}, nil
}
