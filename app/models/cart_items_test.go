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

	// A cart item names a cart and a product, so both have to exist.
	Context("with a cart and a product", func() {
		var cart domain.Cart
		var product domain.Product

		BeforeEach(func() {
			cart = factories.Cart(GinkgoT(), store)
			product = factories.Product(GinkgoT(), store)
		})

		It("creates a cart item", func() {
			created, err := store.CreateCartItem(GinkgoT().Context(), domain.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 2})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeZero())
		})

		Context("with a created cart item", func() {
			var created domain.CartItem
			var id string

			BeforeEach(func() {
				var err error
				created, err = store.CreateCartItem(GinkgoT().Context(), domain.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 2})
				Expect(err).NotTo(HaveOccurred())
				id = strconv.FormatInt(created.ID, 10)
			})

			It("reads it", func() {
				Expect(store.GetCartItem(GinkgoT().Context(), id)).To(Equal(created))
			})

			It("lists it", func() {
				all, err := store.ListCartItems(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(created.ID))
			})

			It("updates it", func() {
				created.Quantity = 5
				updated, err := store.UpdateCartItem(GinkgoT().Context(), created)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Quantity).To(BeEquivalentTo(5))
			})

			It("deletes it", func() {
				Expect(store.DeleteCartItem(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetCartItem(GinkgoT().Context(), id)
				Expect(err).To(MatchError(domain.ErrNotFound))
			})
		})
	})
})
