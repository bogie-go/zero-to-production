package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("Product.Validate", func() {
	var product models.Product

	BeforeEach(func() {
		product = models.Product{Name: "Widget", PriceCents: 500, Stock: 10}
	})

	Context("with a name, a price and stock", func() {
		It("is valid", func() {
			Expect(product.Validate()).To(Succeed())
		})
	})

	Context("with a price of zero and no stock", func() {
		BeforeEach(func() {
			product.PriceCents, product.Stock = 0, 0
		})

		It("is valid", func() {
			Expect(product.Validate()).To(Succeed())
		})
	})

	DescribeTable("is refused for",
		func(change func(*models.Product), reason string) {
			change(&product)

			err := product.Validate()

			Expect(err).To(MatchError(models.ErrInvalid))
			Expect(err).To(MatchError(ContainSubstring(reason)))
		},
		Entry("an empty name", func(p *models.Product) { p.Name = "" }, "name is required"),
		Entry("a name of spaces", func(p *models.Product) { p.Name = "   " }, "name is required"),
		Entry("a negative price", func(p *models.Product) { p.PriceCents = -1 }, "price cannot be negative"),
		Entry("negative stock", func(p *models.Product) { p.Stock = -1 }, "stock cannot be negative"),
	)
})
