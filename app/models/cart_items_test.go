package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
)

var _ = Describe("Cart items", func() {

	DescribeTable("an unknown cart item is not found",
		func(id string) {
			_, err := store.GetCartItem(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteCartItem(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		// A valid-looking line, so Validate passes and the not-found check is
		// what gets exercised.
		_, err := store.UpdateCartItem(GinkgoT().Context(), domain.CartItem{ID: 999999999, CartID: 1, ProductID: 1, Quantity: 1})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	It("is created, read, listed, updated and deleted", func() {
		ctx := GinkgoT().Context()
		// A cart item names a cart and a product, so both have to exist.
		cart := factories.Cart(GinkgoT(), store)
		product := factories.Product(GinkgoT(), store)

		created, err := store.CreateCartItem(ctx, domain.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 2})
		Expect(err).NotTo(HaveOccurred())
		id := strconv.FormatInt(created.ID, 10)

		Expect(store.GetCartItem(ctx, id)).To(Equal(created))
		all, err := store.ListCartItems(ctx, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(all).To(HaveLen(1))
		Expect(all[0].ID).To(Equal(created.ID))

		created.Quantity = 5
		updated, err := store.UpdateCartItem(ctx, created)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Quantity).To(BeEquivalentTo(5))

		Expect(store.DeleteCartItem(ctx, id)).To(Succeed())
		_, err = store.GetCartItem(ctx, id)
		Expect(err).To(MatchError(domain.ErrNotFound))
	})
})
