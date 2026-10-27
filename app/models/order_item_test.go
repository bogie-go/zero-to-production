package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("OrderItem.Validate", func() {
	var item models.OrderItem

	BeforeEach(func() {
		item = models.OrderItem{OrderID: 1, ProductID: 1, Quantity: 2, UnitPriceCents: 500}
	})

	Context("with an order, a product, a quantity and a price", func() {
		It("is valid", func() {
			Expect(item.Validate()).To(Succeed())
		})
	})

	// A free item (a promotion) is a legitimate line; only a negative price
	// is nonsense.
	Context("with a unit price of zero", func() {
		BeforeEach(func() {
			item.UnitPriceCents = 0
		})

		It("is valid", func() {
			Expect(item.Validate()).To(Succeed())
		})
	})

	DescribeTable("is refused for",
		func(change func(*models.OrderItem), reason string) {
			change(&item)

			err := item.Validate()

			Expect(err).To(MatchError(models.ErrInvalid))
			Expect(err).To(MatchError(ContainSubstring(reason)))
		},
		Entry("no order", func(i *models.OrderItem) { i.OrderID = 0 }, "order is required"),
		Entry("no product", func(i *models.OrderItem) { i.ProductID = 0 }, "product is required"),
		Entry("a quantity of zero", func(i *models.OrderItem) { i.Quantity = 0 }, "quantity must be positive"),
		Entry("a negative quantity", func(i *models.OrderItem) { i.Quantity = -1 }, "quantity must be positive"),
		Entry("a negative unit price", func(i *models.OrderItem) { i.UnitPriceCents = -1 }, "unit price cannot be negative"),
	)
})
