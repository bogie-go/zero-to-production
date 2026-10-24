package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/test/factories"
)

// The cross-cart bug from task C2: a caller can never update or delete another
// visitor's line by guessing a bigint id.
var _ = Describe("A cart line", func() {
	var cartA, cartB domain.Cart
	var id string

	BeforeEach(func() {
		ctx := GinkgoT().Context()
		var err error
		cartA, err = store.FindOrCreateCartByToken(ctx, "visitor-a")
		Expect(err).NotTo(HaveOccurred())
		cartB, err = store.FindOrCreateCartByToken(ctx, "visitor-b")
		Expect(err).NotTo(HaveOccurred())
		product := factories.Product(GinkgoT(), store, func(p *domain.Product) { p.Stock = 100 })
		item, err := store.AddCartItem(ctx, cartA.ID, product.ID, 2)
		Expect(err).NotTo(HaveOccurred())
		id = strconv.FormatInt(item.ID, 10)
	})

	Context("asked for through another cart", func() {
		It("cannot be updated or deleted, and is left untouched", func() {
			ctx := GinkgoT().Context()
			_, err := store.UpdateCartItemQuantity(ctx, cartB.ID, id, 5)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteCartItemForCart(ctx, cartB.ID, id)).To(MatchError(domain.ErrNotFound))

			still, err := store.GetCartItem(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(still.Quantity).To(BeEquivalentTo(2))
		})
	})

	Context("asked for through its own cart", func() {
		It("can be updated and deleted", func() {
			ctx := GinkgoT().Context()
			_, err := store.UpdateCartItemQuantity(ctx, cartA.ID, id, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(store.DeleteCartItemForCart(ctx, cartA.ID, id)).To(Succeed())

			_, err = store.GetCartItem(ctx, id)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
	})
})
