package controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("GET /cart", func() {
	It("changes the total once a price rule exists and is active", func() {
		product := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.PriceCents = 1000 })
		factories.PriceRule(GinkgoT(), app.Store) // active, $1.00 off
		user := factories.User(GinkgoT(), app.Store)
		app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).AsUser(user).Do().Status(http.StatusCreated)

		var cart views.CartResponse
		app.Get("/cart").AsUser(user).Do().Status(http.StatusOK).JSON(&cart)

		Expect(cart.SubtotalCents).To(BeEquivalentTo(2000))
		Expect(cart.DiscountCents).To(BeNumerically(">", 0))
		Expect(cart.TotalCents).To(Equal(cart.SubtotalCents - cart.DiscountCents))
	})

	It("changes what the shop charges once a rule is switched on", func() {
		staff := factories.Staff(GinkgoT(), app.Store)
		product := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.PriceCents = 1000 })
		rule := factories.PriceRule(GinkgoT(), app.Store, func(r *domain.PriceRule) { r.Active = false })
		app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().Status(http.StatusCreated)

		var cart views.CartResponse
		app.Get("/cart").AsGuest("g").Do().JSON(&cart)
		Expect(cart.DiscountCents).To(BeZero())

		app.Post("/admin/price_rules/"+strconv.FormatInt(rule.ID, 10)+"/activate", nil).AsUser(staff).Do().Status(http.StatusOK)
		app.Get("/cart").AsGuest("g").Do().JSON(&cart)
		Expect(cart.DiscountCents).To(BeEquivalentTo(100))
		Expect(cart.AppliedRules).To(HaveLen(1))
	})
})
