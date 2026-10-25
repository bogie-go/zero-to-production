package pricing

import (
	"encoding/json"
	"fmt"

	"ecommerce/app/domain"
)

// BuyXGetYRule is "buy 2, get 1 free": in every group of X+Y units of the
// SAME product in the cart, Y are free. It works line by line because the
// discount is Y units at that product's own unit price, which a cart-wide
// total does not carry. Units of different products never make up a set.
//
// Three of a product at $10 under "buy 2, get 1 free" cost $20; four still
// cost $30 (one set), six cost $40. Two cost $20: the free one has to be a
// third unit in the cart.
//
// The free units come off the running total at their face price, like every
// rule that runs after another; put a percentage rule after this one (a
// higher priority number) if the sale should apply to what is left.
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
	group := int64(r.RequiredQuantity) + int64(r.FreeQuantity)
	var freeTotal, discount int64
	var lines []map[string]any
	for _, l := range ctx.Lines {
		free := l.Quantity / group * int64(r.FreeQuantity)
		if free == 0 {
			continue
		}
		freeTotal += free
		discount += free * l.UnitCents
		lines = append(lines, map[string]any{
			"product_id":     l.ProductID,
			"free_items":     free,
			"discount_cents": free * l.UnitCents,
		})
	}
	// Never take more off than is left, like a fixed discount.
	if discount > ctx.CurrentCents {
		discount = ctx.CurrentCents
	}
	if discount == 0 {
		return nil, nil
	}
	ctx.CurrentCents -= discount
	return map[string]any{
		"required_quantity": r.RequiredQuantity,
		"free_quantity":     r.FreeQuantity,
		"free_items":        freeTotal,
		"lines":             lines,
	}, nil
}

func init() {
	RegisterSpec(Spec{
		Kind: "buy_x_get_y", Label: "Buy X, get Y free",
		Help: "In every group of the two numbers added together, the free ones cost nothing.",
		Note: "Counted per product: units of different products never make up a set. Buy 2, get 1 free means the third one is free.",
		Fields: []Field{
			{Name: "required_quantity", Label: "Buy this many", Type: "int"},
			{Name: "free_quantity", Label: "Get this many free", Type: "int"},
		},
	})
}

// Summary says "Buy 2, get 1 free".
func (r BuyXGetYRule) Summary() string {
	return fmt.Sprintf("Buy %d, get %d free", r.RequiredQuantity, r.FreeQuantity)
}
