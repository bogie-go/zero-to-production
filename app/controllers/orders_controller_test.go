package controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testdb"
)

var _ = Describe("POST /orders", func() {
	var product models.Product
	var user models.User

	It("needs a token to check out", func() {
		app.Post("/orders", nil).Do().Status(http.StatusUnauthorized)
	})

	It("refuses a guest with 403", func() {
		app.Post("/orders", nil).AsGuest("someone").Do().Status(http.StatusForbidden)
	})

	Context("when the stock is taken between adding to the cart and paying", func() {
		BeforeEach(func() {
			product = factories.Product(GinkgoT(), app.Store, func(p *models.Product) { p.Stock = 5 })
			user = factories.User(GinkgoT(), app.Store)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 5}).AsUser(user).Do().Status(http.StatusCreated)
			// The cart refuses more than is in stock when adding, so someone
			// else has to buy it afterwards.
			_, err := testdb.Pool(GinkgoT()).Exec(GinkgoT().Context(), "UPDATE products SET stock = 2 WHERE id = $1", product.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("refuses the checkout and changes nothing", func() {
			app.Post("/orders", nil).AsUser(user).Do().Status(http.StatusUnprocessableEntity)

			left, err := app.Store.GetProduct(GinkgoT().Context(), strconv.FormatInt(product.ID, 10))
			Expect(err).NotTo(HaveOccurred())
			Expect(left.Stock).To(BeEquivalentTo(2))
			var orders []views.OrderSummaryResponse
			app.Get("/orders").AsUser(user).Do().Status(http.StatusOK).JSON(&orders)
			Expect(orders).To(BeEmpty())
			Expect(app.Jobs("send_order_confirmation")).To(BeZero())
		})
	})
})

var _ = Describe("POST /orders with a buy 2 get 1 free rule", func() {
	Context("when the cart holds three of a product", func() {
		var user models.User
		var order views.OrderResponse

		BeforeEach(func() {
			product := factories.Product(GinkgoT(), app.Store, func(p *models.Product) { p.PriceCents = 1000 })
			factories.PriceRule(GinkgoT(), app.Store, func(r *models.PriceRule) {
				r.Kind, r.Properties = "buy_x_get_y", []byte(`{"required_quantity":2,"free_quantity":1}`)
			})
			user = factories.User(GinkgoT(), app.Store)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 3}).AsUser(user).Do().Status(http.StatusCreated)
		})

		It("charges what the cart showed: the third is free", func() {
			app.Post("/orders", nil).AsUser(user).Do().Status(http.StatusCreated).JSON(&order)

			Expect(order.SubtotalCents).To(BeEquivalentTo(3000))
			Expect(order.DiscountCents).To(BeEquivalentTo(1000))
			Expect(order.TotalCents).To(BeEquivalentTo(2000))
		})
	})
})

var _ = Describe("GET /orders/:id", func() {
	var ada, grace models.User
	var path string

	Context("with an order placed by a customer", func() {
		BeforeEach(func() {
			product := factories.Product(GinkgoT(), app.Store)
			ada, grace = factories.User(GinkgoT(), app.Store), factories.User(GinkgoT(), app.Store)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsUser(ada).Do().Status(http.StatusCreated)
			var order views.OrderResponse
			app.Post("/orders", nil).AsUser(ada).Do().Status(http.StatusCreated).JSON(&order)
			path = "/orders/" + strconv.FormatInt(order.ID, 10)
		})

		It("shows the order to its customer", func() {
			app.Get(path).AsUser(ada).Do().Status(http.StatusOK)
		})

		It("hides the order from another customer", func() {
			app.Get(path).AsUser(grace).Do().Status(http.StatusNotFound)
		})

		It("needs a token to list orders", func() {
			app.Get("/orders").Do().Status(http.StatusUnauthorized)
		})
	})
})
