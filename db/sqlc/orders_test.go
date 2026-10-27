package db_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("Orders", func() {
	// An order belongs to a user, so one has to be saved first.
	create := func() models.Order {
		user := factories.User(GinkgoT(), store)
		order, err := store.CreateOrder(GinkgoT().Context(), models.Order{UserID: user.ID, Status: "pending", SubtotalCents: 1000, DiscountCents: 100, TotalCents: 900})
		Expect(err).NotTo(HaveOccurred())
		return order
	}

	Describe("GetOrder", func() {
		DescribeTable("is not found for",
			func(id string) {
				_, err := store.GetOrder(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved order", func() {
			var order models.Order

			BeforeEach(func() {
				order = create()
			})

			It("finds the order by id", func() {
				Expect(store.GetOrder(GinkgoT().Context(), strconv.FormatInt(order.ID, 10))).To(Equal(order))
			})
		})
	})

	Describe("ListOrders", func() {
		Context("with a saved order", func() {
			var order models.Order

			BeforeEach(func() {
				order = create()
			})

			It("includes the saved order", func() {
				all, err := store.ListOrders(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(order.ID))
			})
		})
	})

	Describe("CreateOrder", func() {
		Context("with a user", func() {
			var user models.User

			BeforeEach(func() {
				user = factories.User(GinkgoT(), store)
			})

			It("saves the order with an id", func() {
				created, err := store.CreateOrder(GinkgoT().Context(), models.Order{UserID: user.ID, Status: "pending", SubtotalCents: 1000, DiscountCents: 100, TotalCents: 900})
				Expect(err).NotTo(HaveOccurred())
				Expect(created.ID).NotTo(BeZero())
			})
		})
	})

	Describe("UpdateOrder", func() {
		Context("with a saved order", func() {
			var order models.Order

			BeforeEach(func() {
				order = create()
			})

			It("stores the change", func() {
				order.Status = "paid"
				updated, err := store.UpdateOrder(GinkgoT().Context(), order)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Status).To(Equal("paid"))
			})
		})

		Context("when the order is not there", func() {
			It("is not found", func() {
				// A valid-looking order (UserID need not name a real user: no
				// row matches, so the foreign key is never checked), so
				// Validate passes.
				_, err := store.UpdateOrder(GinkgoT().Context(), models.Order{ID: 999999999, UserID: 1, Status: "pending"})
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("DeleteOrder", func() {
		DescribeTable("is not found for",
			func(id string) {
				Expect(store.DeleteOrder(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved order", func() {
			var order models.Order

			BeforeEach(func() {
				order = create()
			})

			It("removes the order", func() {
				id := strconv.FormatInt(order.ID, 10)
				Expect(store.DeleteOrder(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetOrder(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})
})
