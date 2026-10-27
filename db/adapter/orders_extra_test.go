package adapter_test

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
	"ecommerce/test/testdb"
)

var _ = Describe("Checkout (CreateOrderFromCart)", func() {
	var user models.User
	var cart models.Cart
	var product models.Product // $5.00, 10 in stock

	addToCart := func(p models.Product, quantity int32) {
		_, err := store.AddCartItem(GinkgoT().Context(), cart.ID, p.ID, quantity)
		Expect(err).NotTo(HaveOccurred())
	}
	stockOf := func(p models.Product) int32 {
		got, err := store.GetProduct(GinkgoT().Context(), strconv.FormatInt(p.ID, 10))
		Expect(err).NotTo(HaveOccurred())
		return got.Stock
	}

	Context("with a user, an empty cart and a $5.00 product with 10 in stock", func() {
		BeforeEach(func() {
			product = factories.Product(GinkgoT(), store, func(p *models.Product) { p.PriceCents = 500; p.Stock = 10 })
			user = factories.User(GinkgoT(), store)
			cart = factories.Cart(GinkgoT(), store)
		})

		// A successful checkout decrements stock, snapshots the price into an
		// order_item, and empties the cart, all three in one transaction.
		Context("with three of the product in the cart", func() {
			BeforeEach(func() {
				addToCart(product, 3)
			})

			It("decrements stock, snapshots the price and empties the cart", func() {
				order, lines, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)

				Expect(err).NotTo(HaveOccurred())
				Expect(order.UserID).To(Equal(user.ID))
				Expect(order.SubtotalCents).To(BeEquivalentTo(1500))
				Expect(order.TotalCents).To(BeEquivalentTo(1500))
				Expect(lines).To(HaveLen(1))
				Expect(lines[0].ProductID).To(Equal(product.ID))
				Expect(lines[0].Quantity).To(BeEquivalentTo(3))
				Expect(lines[0].UnitPriceCents).To(BeEquivalentTo(500))
				Expect(stockOf(product)).To(BeEquivalentTo(7))
				Expect(store.CartContents(GinkgoT().Context(), cart.ID)).To(BeEmpty())
			})
		})

		// The transaction's whole point: when one line can't be filled, nothing
		// happens. AddCartItem's stock check only looks at add-to-cart time, so by
		// checkout it can be stale.
		Context("with one line that can no longer be filled", func() {
			BeforeEach(func() {
				scarce := factories.Product(GinkgoT(), store, func(p *models.Product) { p.Stock = 1 })
				addToCart(product, 2)
				addToCart(scarce, 1)
				// Someone else buys the last unit between adding and checking out.
				_, err := testdb.Pool(GinkgoT()).Exec(GinkgoT().Context(), "UPDATE products SET stock = 0 WHERE id = $1", scarce.ID)
				Expect(err).NotTo(HaveOccurred())
			})

			It("rolls everything back", func() {
				_, _, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)

				Expect(err).To(MatchError(models.ErrInvalid))
				Expect(stockOf(product)).To(BeEquivalentTo(10), "the line that could be filled is rolled back too")
				Expect(store.CartContents(GinkgoT().Context(), cart.ID)).To(HaveLen(2))
				Expect(store.ListOrdersForUser(GinkgoT().Context(), user.ID, 10)).To(BeEmpty())
			})
		})

		Context("with the cart empty", func() {
			var called bool

			BeforeEach(func() {
				called = false
				store.OnOrderPlaced(func(context.Context, pgx.Tx, models.Order) error { called = true; return nil })
			})

			It("is refused", func() {
				_, _, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)
				Expect(err).To(MatchError(models.ErrInvalid))
			})

			It("does not run the OnOrderPlaced hook", func() {
				_, _, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)

				Expect(err).To(HaveOccurred())
				Expect(called).To(BeFalse())
			})
		})

		Describe("GetOrderForUser", func() {
			Context("with an order placed by the user and another user", func() {
				var other models.User
				var id string

				BeforeEach(func() {
					other = factories.User(GinkgoT(), store)
					addToCart(product, 1)
					order, _, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)
					Expect(err).NotTo(HaveOccurred())
					id = strconv.FormatInt(order.ID, 10)
				})

				It("is not found for another user", func() {
					_, _, err := store.GetOrderForUser(GinkgoT().Context(), other.ID, id)
					Expect(err).To(MatchError(models.ErrNotFound))
				})

				It("finds it for its owner", func() {
					_, _, err := store.GetOrderForUser(GinkgoT().Context(), user.ID, id)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		// The hook runs inside the checkout's transaction, after the order exists,
		// with that transaction: what it writes there commits with the order.
		Describe("the OnOrderPlaced hook", func() {
			Context("with one in the cart and a hook that counts the order inside and outside the transaction", func() {
				var seen models.Order
				var insideCount, outsideCount int

				BeforeEach(func() {
					addToCart(product, 1)
					pool := testdb.Pool(GinkgoT())
					seen, insideCount, outsideCount = models.Order{}, 0, 0
					store.OnOrderPlaced(func(ctx context.Context, tx pgx.Tx, order models.Order) error {
						seen = order
						// Visible on this transaction, not yet outside it.
						if err := tx.QueryRow(ctx, "SELECT count(*) FROM orders WHERE id = $1", order.ID).Scan(&insideCount); err != nil {
							return err
						}
						return pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE id = $1", order.ID).Scan(&outsideCount)
					})
				})

				It("runs inside the transaction", func() {
					order, _, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)

					Expect(err).NotTo(HaveOccurred())
					Expect(seen.ID).To(Equal(order.ID))
					Expect(seen.UserID).To(Equal(user.ID))
					Expect(insideCount).To(Equal(1), "inside the transaction the order is visible")
					Expect(outsideCount).To(BeZero(), "outside it, not before commit")
				})
			})

			// A hook that fails undoes the checkout, which is what makes "the job
			// exists if and only if the order does" true.
			Context("with three in the cart and a hook that fails", func() {
				BeforeEach(func() {
					addToCart(product, 3)
					store.OnOrderPlaced(func(context.Context, pgx.Tx, models.Order) error { return errors.New("queue is down") })
				})

				It("rolls the checkout back", func() {
					_, _, err := store.CreateOrderFromCart(GinkgoT().Context(), user.ID, cart.ID)

					Expect(err).To(HaveOccurred())
					Expect(store.ListOrdersForUser(GinkgoT().Context(), user.ID, 10)).To(BeEmpty())
					Expect(stockOf(product)).To(BeEquivalentTo(10))
					Expect(store.CartContents(GinkgoT().Context(), cart.ID)).To(HaveLen(1))
				})
			})
		})
	})
})
