package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
)

var _ = Describe("Orders", func() {

	DescribeTable("an unknown order is not found",
		func(id string) {
			_, err := store.GetOrder(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteOrder(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		// A valid-looking order (UserID need not name a real user: no row
		// matches, so the foreign key is never checked), so Validate passes.
		_, err := store.UpdateOrder(GinkgoT().Context(), domain.Order{ID: 999999999, UserID: 1, Status: "pending"})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	Context("with a user", func() {
		var user domain.User
		var order domain.Order

		BeforeEach(func() {
			user = factories.User(GinkgoT(), store)
			order = domain.Order{UserID: user.ID, Status: "pending", SubtotalCents: 1000, DiscountCents: 100, TotalCents: 900}
		})

		It("creates an order", func() {
			created, err := store.CreateOrder(GinkgoT().Context(), order)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeZero())
		})

		Context("with a created order", func() {
			var created domain.Order
			var id string

			BeforeEach(func() {
				var err error
				created, err = store.CreateOrder(GinkgoT().Context(), order)
				Expect(err).NotTo(HaveOccurred())
				id = strconv.FormatInt(created.ID, 10)
			})

			It("reads it", func() {
				Expect(store.GetOrder(GinkgoT().Context(), id)).To(Equal(created))
			})

			It("lists it", func() {
				all, err := store.ListOrders(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(created.ID))
			})

			It("updates it", func() {
				created.Status = "paid"
				updated, err := store.UpdateOrder(GinkgoT().Context(), created)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Status).To(Equal("paid"))
			})

			It("deletes it", func() {
				Expect(store.DeleteOrder(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetOrder(GinkgoT().Context(), id)
				Expect(err).To(MatchError(domain.ErrNotFound))
			})
		})
	})
})
