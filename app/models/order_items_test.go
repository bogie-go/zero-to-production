package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
)

var _ = Describe("Order items", func() {

	DescribeTable("an unknown order item is not found",
		func(id string) {
			_, err := store.GetOrderItem(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteOrderItem(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		// OrderID and ProductID need not name real rows: no row matches, so
		// neither foreign key is checked; Validate passes.
		_, err := store.UpdateOrderItem(GinkgoT().Context(), domain.OrderItem{ID: 999999999, OrderID: 1, ProductID: 1, Quantity: 1, UnitPriceCents: 100})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	Context("with an order and a product", func() {
		var item domain.OrderItem

		BeforeEach(func() {
			user := factories.User(GinkgoT(), store)
			order, err := store.CreateOrder(GinkgoT().Context(), domain.Order{UserID: user.ID, Status: "pending"})
			Expect(err).NotTo(HaveOccurred())
			product := factories.Product(GinkgoT(), store)
			item = domain.OrderItem{OrderID: order.ID, ProductID: product.ID, Quantity: 2, UnitPriceCents: 500}
		})

		It("creates an order item", func() {
			created, err := store.CreateOrderItem(GinkgoT().Context(), item)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeZero())
		})

		Context("with a created order item", func() {
			var created domain.OrderItem
			var id string

			BeforeEach(func() {
				var err error
				created, err = store.CreateOrderItem(GinkgoT().Context(), item)
				Expect(err).NotTo(HaveOccurred())
				id = strconv.FormatInt(created.ID, 10)
			})

			It("reads it", func() {
				Expect(store.GetOrderItem(GinkgoT().Context(), id)).To(Equal(created))
			})

			It("lists it", func() {
				all, err := store.ListOrderItems(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(created.ID))
			})

			It("updates it", func() {
				created.Quantity = 3
				updated, err := store.UpdateOrderItem(GinkgoT().Context(), created)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Quantity).To(BeEquivalentTo(3))
			})

			It("deletes it", func() {
				Expect(store.DeleteOrderItem(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetOrderItem(GinkgoT().Context(), id)
				Expect(err).To(MatchError(domain.ErrNotFound))
			})
		})
	})
})
