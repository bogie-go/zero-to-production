package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
)

var _ = Describe("Carts by token", func() {

	Describe("GetCartByToken", func() {
		It("does not create a cart: an unknown token is not found", func() {
			_, err := store.GetCartByToken(GinkgoT().Context(), "nobody")
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		Context("with a cart found or created for a token", func() {
			BeforeEach(func() {
				_, err := store.FindOrCreateCartByToken(GinkgoT().Context(), "nobody")
				Expect(err).NotTo(HaveOccurred())
			})

			It("finds it", func() {
				found, err := store.GetCartByToken(GinkgoT().Context(), "nobody")
				Expect(err).NotTo(HaveOccurred())
				Expect(found.Token).To(Equal("nobody"))
			})
		})
	})

	Describe("DeleteCartAndItems", func() {
		Context("with a cart holding a line", func() {
			var cart domain.Cart

			BeforeEach(func() {
				ctx := GinkgoT().Context()
				var err error
				cart, err = store.FindOrCreateCartByToken(ctx, "guest-abc")
				Expect(err).NotTo(HaveOccurred())
				product := factories.Product(GinkgoT(), store, func(p *domain.Product) { p.Stock = 100 })
				_, err = store.AddCartItem(ctx, cart.ID, product.ID, 2)
				Expect(err).NotTo(HaveOccurred())
			})

			It("removes the cart and its lines", func() {
				ctx := GinkgoT().Context()
				Expect(store.DeleteCartAndItems(ctx, cart.ID)).To(Succeed())

				_, err := store.GetCartByToken(ctx, "guest-abc")
				Expect(err).To(MatchError(domain.ErrNotFound))
				Expect(store.CartContents(ctx, cart.ID)).To(BeEmpty())
			})
		})
	})

	// Two first requests for the same token at the same moment still end up
	// with one cart: the unique index plus the single upsert, not luck.
	Describe("FindOrCreateCartByToken", func() {
		It("is safe under concurrency", func() {
			ctx := GinkgoT().Context()
			const n = 8
			ids := make(chan int64, n)
			errs := make(chan error, n)
			for i := 0; i < n; i++ {
				go func() {
					defer GinkgoRecover()
					c, err := store.FindOrCreateCartByToken(ctx, "racing-guest")
					ids <- c.ID
					errs <- err
				}()
			}

			var seen []int64
			for i := 0; i < n; i++ {
				Expect(<-errs).NotTo(HaveOccurred())
				seen = append(seen, <-ids)
			}
			Expect(seen).To(HaveEach(seen[0]), "every caller gets the same cart")
			Expect(store.ListCarts(ctx, 100)).To(HaveLen(1))
		})
	})

	// The index is the guarantee itself.
	Context("with a cart whose token is taken", func() {
		BeforeEach(func() {
			_, err := store.CreateCart(GinkgoT().Context(), domain.Cart{Token: "same"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("refuses a second cart with the same token", func() {
			_, err := store.CreateCart(GinkgoT().Context(), domain.Cart{Token: "same"})
			Expect(err).To(HaveOccurred())
		})
	})
})
