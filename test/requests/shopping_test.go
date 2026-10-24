package requests_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testdb"
)

var _ = Describe("Shopping", func() {

	stockOf := func(p domain.Product) int32 {
		left, err := app.Store.GetProduct(GinkgoT().Context(), strconv.FormatInt(p.ID, 10))
		Expect(err).NotTo(HaveOccurred())
		return left.Stock
	}

	// The whole customer journey, through HTTP only.
	It("takes a guest from an empty cart, through signup, to a placed order", func() {
		product := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.PriceCents = 500; p.Stock = 10 })

		var guest views.GuestSessionResponse
		app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
		app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 3}).
			WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)

		By("refusing checkout to a guest")
		app.Post("/orders", nil).WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusForbidden)

		By("signing up, which keeps the cart and queues the welcome email")
		var session views.SessionResponse
		app.Post("/users", views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}).
			WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated).JSON(&session)
		Expect(app.Jobs("send_welcome_email")).To(Equal(1))

		var cart views.CartResponse
		app.Get("/cart").WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusOK).JSON(&cart)
		Expect(cart.Items).To(HaveLen(1))
		Expect(cart.Items[0].Quantity).To(BeEquivalentTo(3))
		Expect(cart.TotalCents).To(BeEquivalentTo(1500))

		By("checking out")
		var order views.OrderResponse
		app.Post("/orders", nil).WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusCreated).JSON(&order)
		Expect(order.TotalCents).To(BeEquivalentTo(1500))
		Expect(order.Items).To(HaveLen(1))
		Expect(order.Items[0].UnitPriceCents).To(BeEquivalentTo(500))
		Expect(app.Jobs("send_order_confirmation")).To(Equal(1))
		Expect(stockOf(product)).To(BeEquivalentTo(7))

		app.Get("/cart").WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusOK).JSON(&cart)
		Expect(cart.Items).To(BeEmpty())
	})

	When("the stock is taken between adding to the cart and paying", func() {
		var product domain.Product
		var user domain.User

		BeforeEach(func() {
			product = factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.Stock = 5 })
			user = factories.User(GinkgoT(), app.Store)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 5}).AsUser(user).Do().Status(http.StatusCreated)
			// The cart refuses more than is in stock when adding, so someone
			// else has to buy it afterwards.
			_, err := testdb.Pool(GinkgoT()).Exec(GinkgoT().Context(), "UPDATE products SET stock = 2 WHERE id = $1", product.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("refuses the checkout and changes nothing", func() {
			app.Post("/orders", nil).AsUser(user).Do().Status(http.StatusUnprocessableEntity)

			Expect(stockOf(product)).To(BeEquivalentTo(2))
			var orders []views.OrderSummaryResponse
			app.Get("/orders").AsUser(user).Do().Status(http.StatusOK).JSON(&orders)
			Expect(orders).To(BeEmpty())
			Expect(app.Jobs("send_order_confirmation")).To(BeZero())
		})
	})

	Describe("an order's privacy", func() {
		It("shows a customer only their own orders", func() {
			product := factories.Product(GinkgoT(), app.Store)
			ada, grace := factories.User(GinkgoT(), app.Store), factories.User(GinkgoT(), app.Store)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsUser(ada).Do().Status(http.StatusCreated)
			var order views.OrderResponse
			app.Post("/orders", nil).AsUser(ada).Do().Status(http.StatusCreated).JSON(&order)
			path := "/orders/" + strconv.FormatInt(order.ID, 10)

			app.Get(path).AsUser(ada).Do().Status(http.StatusOK)
			app.Get(path).AsUser(grace).Do().Status(http.StatusNotFound)
			app.Get("/orders").Do().Status(http.StatusUnauthorized)
		})
	})
})
