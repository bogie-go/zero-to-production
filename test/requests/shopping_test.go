package requests_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("Shopping", func() {

	stockOf := func(p models.Product) int32 {
		left, err := app.Store.GetProduct(GinkgoT().Context(), strconv.FormatInt(p.ID, 10))
		Expect(err).NotTo(HaveOccurred())
		return left.Stock
	}

	// The whole customer journey, through HTTP only.
	It("takes a guest from an empty cart, through signup, to a placed order", func() {
		product := factories.Product(GinkgoT(), app.Store, func(p *models.Product) { p.PriceCents = 500; p.Stock = 10 })

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
})
