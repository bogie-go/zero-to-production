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

	It("is created, read, listed, updated and deleted", func() {
		ctx := GinkgoT().Context()
		user := factories.User(GinkgoT(), store)

		created, err := store.CreateOrder(ctx, domain.Order{
			UserID: user.ID, Status: "pending", SubtotalCents: 1000, DiscountCents: 100, TotalCents: 900,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(created.ID).NotTo(BeZero())
		id := strconv.FormatInt(created.ID, 10)

		Expect(store.GetOrder(ctx, id)).To(Equal(created))
		all, err := store.ListOrders(ctx, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(all).To(HaveLen(1))
		Expect(all[0].ID).To(Equal(created.ID))

		created.Status = "paid"
		updated, err := store.UpdateOrder(ctx, created)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Status).To(Equal("paid"))

		Expect(store.DeleteOrder(ctx, id)).To(Succeed())
		_, err = store.GetOrder(ctx, id)
		Expect(err).To(MatchError(domain.ErrNotFound))
	})
})
