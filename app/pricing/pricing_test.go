package pricing_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/pricing"
)

func rule(id int64, name, kind string, properties any) domain.PriceRule {
	b, err := json.Marshal(properties)
	Expect(err).NotTo(HaveOccurred())
	return domain.PriceRule{ID: id, Name: name, Kind: kind, Properties: b, Active: true}
}

var _ = Describe("Calculate", func() {
	DescribeTable("final price and applied rules",
		func(startCents int64, rules []domain.PriceRule, metadata map[string]any, wantFinal int64, wantRules int) {
			result, err := pricing.Calculate(startCents, rules, metadata)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.FinalCents).To(Equal(wantFinal))
			Expect(result.AppliedRules).To(HaveLen(wantRules))
		},
		Entry("no rules leaves the price alone",
			int64(1000), nil, nil, int64(1000), 0),
		Entry("percentage discount",
			int64(1000), []domain.PriceRule{rule(1, "Flash Sale", "percentage_discount", pricing.PercentageDiscountRule{Percentage: 20})}, nil, int64(800), 1),
		Entry("fixed discount",
			int64(1000), []domain.PriceRule{rule(1, "Early Bird", "fixed_discount", pricing.FixedDiscountRule{AmountCents: 50})}, nil, int64(950), 1),
		Entry("fixed discount never goes below zero",
			int64(30), []domain.PriceRule{rule(1, "Early Bird", "fixed_discount", pricing.FixedDiscountRule{AmountCents: 50})}, nil, int64(0), 1),
		Entry("minimum purchase below threshold does not apply",
			int64(1000), []domain.PriceRule{rule(1, "Big Spender", "minimum_purchase", pricing.MinimumPurchaseRule{MinimumCents: 50000, DiscountCents: 7500})}, nil, int64(1000), 0),
		Entry("minimum purchase at threshold applies",
			int64(50000), []domain.PriceRule{rule(1, "Big Spender", "minimum_purchase", pricing.MinimumPurchaseRule{MinimumCents: 50000, DiscountCents: 7500})}, nil, int64(42500), 1),
		Entry("buy x get y below required quantity does not apply",
			int64(1000), []domain.PriceRule{rule(1, "BOGO", "buy_x_get_y", pricing.BuyXGetYRule{RequiredQuantity: 2, FreeQuantity: 1})}, map[string]any{"quantity": 1}, int64(1000), 0),
		Entry("buy x get y at required quantity applies",
			int64(1000), []domain.PriceRule{rule(1, "BOGO", "buy_x_get_y", pricing.BuyXGetYRule{RequiredQuantity: 2, FreeQuantity: 1})}, map[string]any{"quantity": 5}, int64(1000), 1),
		// 1000 -> 800 (20% off) -> 725 (meets the $5 minimum, -$0.75)
		Entry("stacking two rules in order",
			int64(1000), []domain.PriceRule{
				rule(1, "Flash Sale", "percentage_discount", pricing.PercentageDiscountRule{Percentage: 20}),
				rule(2, "Big Spender", "minimum_purchase", pricing.MinimumPurchaseRule{MinimumCents: 500, DiscountCents: 75}),
			}, nil, int64(725), 2),
		// 1000 -> 950 -> 760
		Entry("order changes the result: fixed then percentage",
			int64(1000), []domain.PriceRule{
				rule(1, "Early Bird", "fixed_discount", pricing.FixedDiscountRule{AmountCents: 50}),
				rule(2, "Flash Sale", "percentage_discount", pricing.PercentageDiscountRule{Percentage: 20}),
			}, nil, int64(760), 2),
	)

	Context("with a rule of an unknown kind", func() {
		var rules []domain.PriceRule

		BeforeEach(func() {
			rules = []domain.PriceRule{rule(1, "Mystery", "no_such_kind", map[string]any{})}
		})

		It("treats it as invalid", func() {
			_, err := pricing.Calculate(1000, rules, nil)
			Expect(err).To(MatchError(domain.ErrInvalid))
		})
	})
})

var _ = Describe("Result", func() {
	var (
		result *pricing.Result
		err    error
	)

	Context("after a 25% discount on 1000 cents", func() {
		BeforeEach(func() {
			result, err = pricing.Calculate(1000, []domain.PriceRule{rule(1, "Flash Sale", "percentage_discount", pricing.PercentageDiscountRule{Percentage: 25})}, nil)
		})

		It("reports the savings", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalSavingsCents()).To(BeEquivalentTo(250))
			Expect(result.SavingsPercentage()).To(BeNumerically("==", 25))
		})
	})
})

var _ = Describe("Load", func() {
	DescribeTable("refuses invalid properties",
		func(kind string, properties any) {
			b, err := json.Marshal(properties)
			Expect(err).NotTo(HaveOccurred())
			_, err = pricing.Load(kind, b)
			Expect(err).To(MatchError(domain.ErrInvalid))
		},
		Entry("negative fixed amount", "fixed_discount", pricing.FixedDiscountRule{AmountCents: -1}),
		Entry("zero percentage", "percentage_discount", pricing.PercentageDiscountRule{Percentage: 0}),
		Entry("percentage over 100", "percentage_discount", pricing.PercentageDiscountRule{Percentage: 101}),
		Entry("zero required quantity", "buy_x_get_y", pricing.BuyXGetYRule{RequiredQuantity: 0, FreeQuantity: 1}),
		Entry("negative minimum", "minimum_purchase", pricing.MinimumPurchaseRule{MinimumCents: -1}),
	)
})

// Every registered kind describes itself: a label, and fields that name real
// JSON keys of its properties, so the admin form can never ask for a field the
// decoder does not read.
var _ = Describe("Specs", func() {
	It("has one per registered kind", func() {
		var specKinds []string
		for _, s := range pricing.Specs() {
			specKinds = append(specKinds, s.Kind)
		}
		Expect(specKinds).To(ConsistOf(pricing.RegisteredKinds()))
	})

	It("gives every kind a label and fields that are its properties", func() {
		for _, s := range pricing.Specs() {
			Expect(s.Label).NotTo(BeEmpty(), s.Kind)
			Expect(s.Fields).NotTo(BeEmpty(), s.Kind)

			// Feed the decoder an object holding exactly the spec's fields: if it
			// reads a different name, the properties come back zero and invalid.
			props := map[string]any{}
			for _, f := range s.Fields {
				props[f.Name] = 5
			}
			raw, err := json.Marshal(props)
			Expect(err).NotTo(HaveOccurred())
			_, err = pricing.Load(s.Kind, raw)
			Expect(err).NotTo(HaveOccurred(), "%s: properties built from the spec's fields (%s) do not load", s.Kind, raw)
		}
	})
})

var _ = Describe("Summarize", func() {
	DescribeTable("says what a rule does",
		func(kind, props, want string) {
			Expect(pricing.Summarize(kind, json.RawMessage(props))).To(Equal(want))
		},
		Entry("fixed discount", "fixed_discount", `{"amount_cents": 5000}`, "$50.00 off"),
		Entry("percentage discount", "percentage_discount", `{"percentage": 20}`, "20% off"),
		Entry("fractional percentage", "percentage_discount", `{"percentage": 12.5}`, "12.5% off"),
		Entry("minimum purchase", "minimum_purchase", `{"minimum_cents": 50000, "discount_cents": 7500}`, "$75.00 off orders of $500.00 or more"),
		Entry("buy x get y", "buy_x_get_y", `{"required_quantity": 2, "free_quantity": 1}`, "Buy 2, get 1 free"),
		Entry("unknown kind", "no_such_kind", `{}`, "invalid no_such_kind rule"),
		Entry("invalid properties", "percentage_discount", `{"percentage": 500}`, "invalid percentage_discount rule"),
	)
})
