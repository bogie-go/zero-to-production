package pricing

import (
	"fmt"

	"ecommerce/app/domain"
)

// Result is a calculation's outcome: the price before and after every rule,
// what ran, and the savings — the same role Rails' Result object plays.
type Result struct {
	OriginalCents int64
	FinalCents    int64
	AppliedRules  []AppliedRule
	Metadata      map[string]any
}

// TotalSavingsCents is how much the applied rules took off.
func (r *Result) TotalSavingsCents() int64 { return r.OriginalCents - r.FinalCents }

// SavingsPercentage is TotalSavingsCents as a percentage of OriginalCents, 0
// if there was nothing to discount from.
func (r *Result) SavingsPercentage() float64 {
	if r.OriginalCents == 0 {
		return 0
	}
	return float64(r.TotalSavingsCents()) / float64(r.OriginalCents) * 100
}

// Calculate runs rules in order against cents, the same pipeline the Rails
// version runs: each rule sees the price the last one left behind. Rules are
// domain.PriceRule rows, not already-decoded Rules, so a caller can load
// them straight from the store (ordered by priority, already filtered to
// active and in-date) without decoding anything itself.
func Calculate(cents int64, rules []domain.PriceRule, metadata map[string]any) (*Result, error) {
	return run(NewPriceContext(cents, metadata), rules)
}

// CalculateLines prices a cart: the subtotal is the sum of its lines, and
// every rule also gets the lines themselves, so a rule that discounts a
// particular product (buy 2, get 1 free) knows what that product costs.
// Rules still run in the order given, each on what the last one left.
func CalculateLines(lines []Line, rules []domain.PriceRule) (*Result, error) {
	var subtotal int64
	for _, l := range lines {
		subtotal += l.UnitCents * l.Quantity
	}
	ctx := NewPriceContext(subtotal, nil)
	ctx.Lines = lines
	return run(ctx, rules)
}

// CartLines turns what the store returns for a cart into the lines the rules
// price, so the cart and the checkout feed the engine the same way.
func CartLines(cart []domain.CartLine) []Line {
	lines := make([]Line, 0, len(cart))
	for _, l := range cart {
		lines = append(lines, Line{ProductID: l.ProductID, UnitCents: int64(l.UnitPriceCents), Quantity: int64(l.Quantity)})
	}
	return lines
}

func run(ctx *PriceContext, rules []domain.PriceRule) (*Result, error) {
	for _, row := range rules {
		rule, err := Load(row.Kind, row.Properties)
		if err != nil {
			return nil, fmt.Errorf("pricing: rule %d %q: %w", row.ID, row.Name, err)
		}
		previous := ctx.CurrentCents
		detail, err := rule.Apply(ctx)
		if err != nil {
			return nil, fmt.Errorf("pricing: apply rule %d %q: %w", row.ID, row.Name, err)
		}
		if detail == nil {
			continue
		}
		ctx.AppliedRules = append(ctx.AppliedRules, AppliedRule{
			Name: row.Name, Kind: row.Kind, PreviousCents: previous, NewCents: ctx.CurrentCents, Detail: detail,
		})
	}
	return &Result{
		OriginalCents: ctx.OriginalCents,
		FinalCents:    ctx.CurrentCents,
		AppliedRules:  ctx.AppliedRules,
		Metadata:      ctx.Metadata,
	}, nil
}
