package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("Order.Validate", func() {
	var order models.Order

	BeforeEach(func() {
		order = models.Order{UserID: 1, Status: "pending", SubtotalCents: 1000, DiscountCents: 100, TotalCents: 900}
	})

	Context("with a user, a known status and amounts", func() {
		It("is valid", func() {
			Expect(order.Validate()).To(Succeed())
		})
	})

	Context("with amounts of zero", func() {
		BeforeEach(func() {
			order.SubtotalCents, order.DiscountCents, order.TotalCents = 0, 0, 0
		})

		It("is valid", func() {
			Expect(order.Validate()).To(Succeed())
		})
	})

	DescribeTable("accepts the status",
		func(status string) {
			order.Status = status

			Expect(order.Validate()).To(Succeed())
		},
		Entry("pending", "pending"),
		Entry("paid", "paid"),
		Entry("cancelled", "cancelled"),
	)

	DescribeTable("is refused for",
		func(change func(*models.Order), reason string) {
			change(&order)

			err := order.Validate()

			Expect(err).To(MatchError(models.ErrInvalid))
			Expect(err).To(MatchError(ContainSubstring(reason)))
		},
		Entry("no user", func(o *models.Order) { o.UserID = 0 }, "user is required"),
		Entry("an empty status", func(o *models.Order) { o.Status = "" }, "is not recognized"),
		Entry("a status nobody defined", func(o *models.Order) { o.Status = "shipped" }, `status "shipped" is not recognized`),
		Entry("a negative subtotal", func(o *models.Order) { o.SubtotalCents = -1 }, "amounts cannot be negative"),
		Entry("a negative discount", func(o *models.Order) { o.DiscountCents = -1 }, "amounts cannot be negative"),
		Entry("a negative total", func(o *models.Order) { o.TotalCents = -1 }, "amounts cannot be negative"),
	)
})
