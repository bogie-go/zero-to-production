package models_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("PriceRule.Validate", func() {
	var rule models.PriceRule
	start := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

	BeforeEach(func() {
		rule = models.PriceRule{Name: "Sale", Kind: "percentage_discount", Properties: []byte(`{}`)}
	})

	Context("with a name and a kind", func() {
		It("is valid", func() {
			Expect(rule.Validate()).To(Succeed())
		})
	})

	// Either end of the window may be open: a rule that starts and never ends,
	// or has always applied until a date.
	Context("with only a start", func() {
		BeforeEach(func() {
			rule.StartsAt = &start
		})

		It("is valid", func() {
			Expect(rule.Validate()).To(Succeed())
		})
	})

	Context("with only an end", func() {
		BeforeEach(func() {
			rule.EndsAt = &start
		})

		It("is valid", func() {
			Expect(rule.Validate()).To(Succeed())
		})
	})

	Context("with an end after its start", func() {
		BeforeEach(func() {
			end := start.Add(24 * time.Hour)
			rule.StartsAt, rule.EndsAt = &start, &end
		})

		It("is valid", func() {
			Expect(rule.Validate()).To(Succeed())
		})
	})

	DescribeTable("is refused for",
		func(change func(*models.PriceRule), reason string) {
			change(&rule)

			err := rule.Validate()

			Expect(err).To(MatchError(models.ErrInvalid))
			Expect(err).To(MatchError(ContainSubstring(reason)))
		},
		Entry("an empty name", func(r *models.PriceRule) { r.Name = "" }, "name is required"),
		Entry("a name of spaces", func(r *models.PriceRule) { r.Name = "  " }, "name is required"),
		Entry("an empty kind", func(r *models.PriceRule) { r.Kind = "" }, "kind is required"),
		Entry("an end before its start", func(r *models.PriceRule) {
			before := start.Add(-time.Hour)
			r.StartsAt, r.EndsAt = &start, &before
		}, "it must end after it starts"),
		Entry("an end at the moment it starts", func(r *models.PriceRule) {
			same := start
			r.StartsAt, r.EndsAt = &start, &same
		}, "it must end after it starts"),
	)
})
