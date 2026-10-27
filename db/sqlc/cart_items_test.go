package db_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("CartItems", func() {
	// A cart item names a cart and a product, so both have to exist.
	create := func() models.CartItem {
		cart := factories.Cart(GinkgoT(), store)
		product := factories.Product(GinkgoT(), store)
		item, err := store.CreateCartItem(GinkgoT().Context(), models.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 2})
		Expect(err).NotTo(HaveOccurred())
		return item
	}

	Describe("GetCartItem", func() {
		DescribeTable("is not found for",
			func(id string) {
				_, err := store.GetCartItem(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved cart item", func() {
			var item models.CartItem

			BeforeEach(func() {
				item = create()
			})

			It("finds the cart item by id", func() {
				Expect(store.GetCartItem(GinkgoT().Context(), strconv.FormatInt(item.ID, 10))).To(Equal(item))
			})
		})
	})

	Describe("ListCartItems", func() {
		Context("with a saved cart item", func() {
			var item models.CartItem

			BeforeEach(func() {
				item = create()
			})

			It("includes the saved cart item", func() {
				all, err := store.ListCartItems(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(item.ID))
			})
		})
	})

	Describe("CreateCartItem", func() {
		Context("with a cart and a product", func() {
			var cart models.Cart
			var product models.Product

			BeforeEach(func() {
				cart = factories.Cart(GinkgoT(), store)
				product = factories.Product(GinkgoT(), store)
			})

			It("saves the cart item with an id", func() {
				created, err := store.CreateCartItem(GinkgoT().Context(), models.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 2})
				Expect(err).NotTo(HaveOccurred())
				Expect(created.ID).NotTo(BeZero())
			})
		})
	})

	Describe("UpdateCartItem", func() {
		Context("with a saved cart item", func() {
			var item models.CartItem

			BeforeEach(func() {
				item = create()
			})

			It("stores the change", func() {
				item.Quantity = 5
				updated, err := store.UpdateCartItem(GinkgoT().Context(), item)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Quantity).To(BeEquivalentTo(5))
			})
		})

		Context("when the cart item is not there", func() {
			It("is not found", func() {
				// A valid-looking line, so Validate passes and the not-found
				// check is what gets exercised.
				_, err := store.UpdateCartItem(GinkgoT().Context(), models.CartItem{ID: 999999999, CartID: 1, ProductID: 1, Quantity: 1})
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("DeleteCartItem", func() {
		DescribeTable("is not found for",
			func(id string) {
				Expect(store.DeleteCartItem(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved cart item", func() {
			var item models.CartItem

			BeforeEach(func() {
				item = create()
			})

			It("removes the cart item", func() {
				id := strconv.FormatInt(item.ID, 10)
				Expect(store.DeleteCartItem(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetCartItem(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})
})
