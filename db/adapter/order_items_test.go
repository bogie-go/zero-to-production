package adapter_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("OrderItems", func() {
	// An order item names an order and a product, so both have to exist.
	line := func() models.OrderItem {
		user := factories.User(GinkgoT(), store)
		order, err := store.CreateOrder(GinkgoT().Context(), models.Order{UserID: user.ID, Status: "pending"})
		Expect(err).NotTo(HaveOccurred())
		product := factories.Product(GinkgoT(), store)
		return models.OrderItem{OrderID: order.ID, ProductID: product.ID, Quantity: 2, UnitPriceCents: 500}
	}
	create := func() models.OrderItem {
		item, err := store.CreateOrderItem(GinkgoT().Context(), line())
		Expect(err).NotTo(HaveOccurred())
		return item
	}

	Describe("GetOrderItem", func() {
		DescribeTable("returns ErrNotFound",
			func(id string) {
				_, err := store.GetOrderItem(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("when the id is not a number", "nope"),
			Entry("when the id is empty", ""),
			Entry("when no order item has that id", "999999999"),
		)

		Context("with a saved order item", func() {
			var item models.OrderItem

			BeforeEach(func() {
				item = create()
			})

			It("finds the order item by id", func() {
				Expect(store.GetOrderItem(GinkgoT().Context(), strconv.FormatInt(item.ID, 10))).To(Equal(item))
			})
		})
	})

	Describe("ListOrderItems", func() {
		Context("with a saved order item", func() {
			var item models.OrderItem

			BeforeEach(func() {
				item = create()
			})

			It("includes the saved order item", func() {
				all, err := store.ListOrderItems(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(item.ID))
			})
		})
	})

	Describe("CreateOrderItem", func() {
		Context("with an order and a product", func() {
			var item models.OrderItem

			BeforeEach(func() {
				item = line()
			})

			It("saves the order item with an id", func() {
				created, err := store.CreateOrderItem(GinkgoT().Context(), item)
				Expect(err).NotTo(HaveOccurred())
				Expect(created.ID).NotTo(BeZero())
			})
		})
	})

	Describe("UpdateOrderItem", func() {
		Context("with a saved order item", func() {
			var item models.OrderItem

			BeforeEach(func() {
				item = create()
			})

			It("stores the change", func() {
				item.Quantity = 3
				updated, err := store.UpdateOrderItem(GinkgoT().Context(), item)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Quantity).To(BeEquivalentTo(3))
			})
		})

		Context("when the order item is not there", func() {
			It("is not found", func() {
				// OrderID and ProductID need not name real rows: no row
				// matches, so neither foreign key is checked; Validate passes.
				_, err := store.UpdateOrderItem(GinkgoT().Context(), models.OrderItem{ID: 999999999, OrderID: 1, ProductID: 1, Quantity: 1, UnitPriceCents: 100})
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("DeleteOrderItem", func() {
		DescribeTable("returns ErrNotFound",
			func(id string) {
				Expect(store.DeleteOrderItem(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("when the id is not a number", "nope"),
			Entry("when the id is empty", ""),
			Entry("when no order item has that id", "999999999"),
		)

		Context("with a saved order item", func() {
			var item models.OrderItem

			BeforeEach(func() {
				item = create()
			})

			It("removes the order item", func() {
				id := strconv.FormatInt(item.ID, 10)
				Expect(store.DeleteOrderItem(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetOrderItem(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})
})
