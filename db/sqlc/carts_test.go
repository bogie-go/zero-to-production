package db_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("Carts", func() {
	// Whatever the id looks like, a cart that is not there is ErrNotFound, and
	// a caller cannot tell a malformed id from one that names nothing.
	Describe("GetCart", func() {
		DescribeTable("is not found for",
			func(id string) {
				_, err := store.GetCart(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved cart", func() {
			var cart models.Cart

			BeforeEach(func() {
				cart = factories.Cart(GinkgoT(), store)
			})

			It("finds the cart by id", func() {
				Expect(store.GetCart(GinkgoT().Context(), strconv.FormatInt(cart.ID, 10))).To(Equal(cart))
			})
		})
	})

	Describe("ListCarts", func() {
		Context("with a saved cart", func() {
			var cart models.Cart

			BeforeEach(func() {
				cart = factories.Cart(GinkgoT(), store)
			})

			It("includes the saved cart", func() {
				all, err := store.ListCarts(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(cart.ID))
			})
		})
	})

	Describe("CreateCart", func() {
		Context("with a visitor's token", func() {
			It("saves the cart with an id", func() {
				created, err := store.CreateCart(GinkgoT().Context(), models.Cart{Token: "visitor-1"})
				Expect(err).NotTo(HaveOccurred())
				Expect(created.ID).NotTo(BeZero())
			})
		})
	})

	Describe("UpdateCart", func() {
		Context("with a saved cart", func() {
			var cart models.Cart

			BeforeEach(func() {
				cart = factories.Cart(GinkgoT(), store)
			})

			It("stores the change", func() {
				cart.Token = "visitor-2"
				updated, err := store.UpdateCart(GinkgoT().Context(), cart)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Token).To(Equal("visitor-2"))
			})
		})

		Context("when the cart is not there", func() {
			It("is not found", func() {
				_, err := store.UpdateCart(GinkgoT().Context(), models.Cart{ID: 999999999})
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("DeleteCart", func() {
		DescribeTable("is not found for",
			func(id string) {
				Expect(store.DeleteCart(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved cart", func() {
			var cart models.Cart

			BeforeEach(func() {
				cart = factories.Cart(GinkgoT(), store)
			})

			It("removes the cart", func() {
				id := strconv.FormatInt(cart.ID, 10)
				Expect(store.DeleteCart(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetCart(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})
})
