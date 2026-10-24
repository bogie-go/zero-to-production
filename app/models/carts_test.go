package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
)

var _ = Describe("Carts", func() {

	// Whatever the id looks like, a cart that is not there is ErrNotFound, and
	// a caller cannot tell a malformed id from one that names nothing.
	DescribeTable("an unknown cart is not found",
		func(id string) {
			_, err := store.GetCart(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteCart(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		_, err := store.UpdateCart(GinkgoT().Context(), domain.Cart{ID: 999999999})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	It("is created, read, listed, updated and deleted", func() {
		ctx := GinkgoT().Context()

		created, err := store.CreateCart(ctx, domain.Cart{Token: "visitor-1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(created.ID).NotTo(BeZero())
		id := strconv.FormatInt(created.ID, 10)

		Expect(store.GetCart(ctx, id)).To(Equal(created))
		all, err := store.ListCarts(ctx, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(all).To(HaveLen(1))
		Expect(all[0].ID).To(Equal(created.ID))

		created.Token = "visitor-2"
		updated, err := store.UpdateCart(ctx, created)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Token).To(Equal("visitor-2"))

		Expect(store.DeleteCart(ctx, id)).To(Succeed())
		_, err = store.GetCart(ctx, id)
		Expect(err).To(MatchError(domain.ErrNotFound))
	})
})
