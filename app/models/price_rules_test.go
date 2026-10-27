package models_test

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
)

func ruleNames(rules []domain.PriceRule) []string {
	out := make([]string, len(rules))
	for i, r := range rules {
		out[i] = r.Name
	}
	return out
}

var _ = Describe("PriceRules", func() {

	Describe("GetPriceRule", func() {
		DescribeTable("is not found for",
			func(id string) {
				_, err := store.GetPriceRule(GinkgoT().Context(), id)
				Expect(err).To(MatchError(domain.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)
	})

	Describe("UpdatePriceRule", func() {
		Context("when the rule is not there", func() {
			It("is not found", func() {
				_, err := store.UpdatePriceRule(GinkgoT().Context(), domain.PriceRule{ID: 999999999, Name: "Nope", Kind: "fixed_discount", Properties: []byte(`{}`)})
				Expect(err).To(MatchError(domain.ErrNotFound))
			})
		})
	})

	Describe("DeletePriceRule", func() {
		DescribeTable("is not found for",
			func(id string) {
				Expect(store.DeletePriceRule(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)
	})

	Describe("listing", func() {
		// The admin's list is every rule in the order the engine runs them,
		// active or not; the cart's own list is only the active ones.
		Context("with a second, an off and a first rule", func() {
			BeforeEach(func() {
				factories.PriceRule(GinkgoT(), store, func(r *domain.PriceRule) { r.Name = "second"; r.Priority = 20 })
				factories.PriceRule(GinkgoT(), store, func(r *domain.PriceRule) { r.Name = "off"; r.Priority = 5; r.Active = false })
				factories.PriceRule(GinkgoT(), store, func(r *domain.PriceRule) { r.Name = "first"; r.Priority = 10 })
			})

			It("gives the admin all rules, by priority", func() {
				all, err := store.ListAllPriceRules(GinkgoT().Context())
				Expect(err).NotTo(HaveOccurred())
				Expect(ruleNames(all)).To(Equal([]string{"off", "first", "second"}))
			})

			It("gives the cart the active ones, by priority", func() {
				active, err := store.ListActivePriceRules(GinkgoT().Context())
				Expect(err).NotTo(HaveOccurred())
				Expect(ruleNames(active)).To(Equal([]string{"first", "second"}))
			})
		})
	})

	Describe("switching on and off", func() {
		// Switching a rule off changes nothing but whether it applies: the very
		// next read of the active rules no longer has it.
		Context("with a rule named Sale", func() {
			var id string

			BeforeEach(func() {
				rule := factories.PriceRule(GinkgoT(), store, func(r *domain.PriceRule) { r.Name = "Sale" })
				id = strconv.FormatInt(rule.ID, 10)
			})

			It("takes effect on the next read when switched off", func() {
				ctx := GinkgoT().Context()
				off, err := store.SetPriceRuleActive(ctx, id, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(off.Active).To(BeFalse())
				Expect(off.Name).To(Equal("Sale"))
				Expect(store.ListActivePriceRules(ctx)).To(BeEmpty())
			})

			Context("when it was switched off", func() {
				BeforeEach(func() {
					_, err := store.SetPriceRuleActive(GinkgoT().Context(), id, false)
					Expect(err).NotTo(HaveOccurred())
				})

				It("takes effect on the next read when switched on again", func() {
					ctx := GinkgoT().Context()
					_, err := store.SetPriceRuleActive(ctx, id, true)
					Expect(err).NotTo(HaveOccurred())
					Expect(store.ListActivePriceRules(ctx)).To(HaveLen(1))
				})
			})
		})

		It("is not found for an unknown rule", func() {
			for _, bad := range []string{"nope", "", "999999999"} {
				_, err := store.SetPriceRuleActive(GinkgoT().Context(), bad, true)
				Expect(err).To(MatchError(domain.ErrNotFound), "id %q", bad)
			}
		})
	})

	Describe("validation on save", func() {
		start := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
		before := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

		DescribeTable("refuses a rule",
			func(rule domain.PriceRule) {
				_, err := store.CreatePriceRule(GinkgoT().Context(), rule)
				Expect(err).To(MatchError(domain.ErrInvalid))
			},
			Entry("with no name", domain.PriceRule{Kind: "fixed_discount", Properties: []byte(`{}`)}),
			Entry("with no kind", domain.PriceRule{Name: "x", Properties: []byte(`{}`)}),
			Entry("that ends before it starts", domain.PriceRule{Name: "x", Kind: "fixed_discount", Properties: []byte(`{}`), StartsAt: &start, EndsAt: &before}),
		)
	})

	// The cart's rules are the active ones inside their dates: a rule that has
	// not started, or has ended, is left out even though it is switched on.
	Describe("the active rules and their dates", func() {
		Context("with rules inside and outside their windows", func() {
			BeforeEach(func() {
				hour := time.Hour
				past, future := time.Now().Add(-48*hour), time.Now().Add(48*hour)
				earlier, later := time.Now().Add(-24*hour), time.Now().Add(24*hour)
				rule := func(name string, priority int32, starts, ends *time.Time) {
					factories.PriceRule(GinkgoT(), store, func(r *domain.PriceRule) {
						r.Name, r.Priority, r.StartsAt, r.EndsAt = name, priority, starts, ends
					})
				}
				rule("no dates", 10, nil, nil)
				rule("inside its window", 20, &past, &future)
				rule("started, no end", 30, &earlier, nil)
				rule("ends later, no start", 40, nil, &later)
				rule("not started yet", 50, &later, &future)
				rule("already ended", 60, &past, &earlier)
				rule("ended, no start", 70, nil, &earlier)
			})

			It("leaves out rules outside their window", func() {
				active, err := store.ListActivePriceRules(GinkgoT().Context())
				Expect(err).NotTo(HaveOccurred())
				Expect(ruleNames(active)).To(Equal([]string{"no dates", "inside its window", "started, no end", "ends later, no start"}))
			})
		})
	})
})
