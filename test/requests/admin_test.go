package requests_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("/admin", func() {

	It("is for staff only", func() {
		customer, staff := factories.User(GinkgoT(), app.Store), factories.Staff(GinkgoT(), app.Store)

		app.Get("/admin/products").Do().Status(http.StatusUnauthorized)
		app.Get("/admin/products").AsUser(customer).Do().Status(http.StatusForbidden)
		app.Get("/admin/products").AsGuest("someone").Do().Status(http.StatusForbidden)
		app.Get("/admin/products").AsUser(staff).Do().Status(http.StatusOK)
	})

	Describe("products", func() {
		var staff domain.User

		BeforeEach(func() {
			staff = factories.Staff(GinkgoT(), app.Store)
		})

		It("are created, archived off the shop, and still listed for staff", func() {
			category := factories.Category(GinkgoT(), app.Store)

			var created views.ProductResponse
			app.Post("/admin/products", views.ProductRequest{
				Name: "Desk Lamp", Description: "warm light", PriceCents: 2500, Stock: 4, ImageUrl: "/lamp.webp", CategoryID: &category.ID,
			}).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
			id := strconv.FormatInt(created.ID, 10)

			app.Post("/admin/products", views.ProductRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)

			app.Get("/products/" + id).Do().Status(http.StatusOK).Contains("Desk Lamp")
			app.Post("/admin/products/"+id+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK)
			app.Get("/products/" + id).Do().Status(http.StatusNotFound)

			var all views.ProductsResponse
			app.Get("/admin/products").AsUser(staff).Do().JSON(&all)
			Expect(all.Products).To(HaveLen(1))
			Expect(all.Products[0].ArchivedAt).NotTo(BeNil())

			app.Get("/admin/products?include_archived=false").AsUser(staff).Do().JSON(&all)
			Expect(all.Products).To(BeEmpty())
		})
	})

	Describe("a price rule", func() {
		It("changes the cart total once it exists and is active", func() {
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
	})
})
