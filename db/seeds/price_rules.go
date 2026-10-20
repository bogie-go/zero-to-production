package seeds

import (
	"context"
	"encoding/json"

	"ecommerce/app/domain"
	"ecommerce/app/models"
	"ecommerce/app/pricing"
)

// priceRule builds a domain.PriceRule from one of pricing's own rule
// structs, so a seed's properties can never drift out of sync with what
// pricing.Load actually expects to decode.
func priceRule(name, kind string, priority int32, properties any) domain.PriceRule {
	b, err := json.Marshal(properties)
	if err != nil {
		panic(err) // a seed's own struct literal failing to marshal is a bug, not bad input
	}
	return domain.PriceRule{Name: name, Kind: kind, Properties: b, Active: true, Priority: priority}
}

// priceRules is every promotion SeedPriceRules inserts, adapted from the
// four example rules in "Building an E-commerce Pricing Rules Engine with
// Single Table Inheritance in Ruby on Rails": same four kinds, cents
// instead of dollars. No admin endpoint exists yet to create these through
// the API (docs/book-plan.md defers that to the admin namespace task), so
// for now this list is the only way a rule enters the database.
var priceRules = []domain.PriceRule{
	priceRule("Flash Sale - 20% Off", "percentage_discount", 10, pricing.PercentageDiscountRule{Percentage: 20}),
	priceRule("Early Bird Special", "fixed_discount", 20, pricing.FixedDiscountRule{AmountCents: 5000}),
	priceRule("Buy 2 Get 1 Free", "buy_x_get_y", 30, pricing.BuyXGetYRule{RequiredQuantity: 2, FreeQuantity: 1}),
	priceRule("Big Spender Bonus", "minimum_purchase", 40, pricing.MinimumPurchaseRule{MinimumCents: 50000, DiscountCents: 7500}),
}

// SeedPriceRules seeds price_rules. db seed can run this any number of
// times: FindOrCreatePriceRuleByName leaves anything already there alone.
func SeedPriceRules(ctx context.Context, s *models.Store) error {
	for _, p := range priceRules {
		if _, err := s.FindOrCreatePriceRuleByName(ctx, p); err != nil {
			return err
		}
	}
	return nil
}
