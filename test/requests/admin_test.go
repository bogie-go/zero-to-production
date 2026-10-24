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

	Describe("products, one at a time", func() {
		var staff domain.User
		var product domain.Product
		var path string

		BeforeEach(func() {
			staff = factories.Staff(GinkgoT(), app.Store)
			product = factories.Product(GinkgoT(), app.Store)
			path = "/admin/products/" + strconv.FormatInt(product.ID, 10)
		})

		It("shows, updates and deletes one", func() {
			var shown views.ProductResponse
			app.Get(path).AsUser(staff).Do().Status(http.StatusOK).JSON(&shown)
			Expect(shown.ID).To(Equal(product.ID))

			var updated views.ProductResponse
			app.Patch(path, views.ProductRequest{Name: "Renamed", PriceCents: 1234, Stock: 3, ImageUrl: "/r.webp"}).
				AsUser(staff).Do().Status(http.StatusOK).JSON(&updated)
			Expect(updated.ID).To(Equal(product.ID), "the id is carried over")
			Expect(updated.Name).To(Equal("Renamed"))
			Expect(updated.PriceCents).To(BeEquivalentTo(1234))

			app.Delete(path).AsUser(staff).Do().Status(http.StatusNoContent)
			app.Get(path).AsUser(staff).Do().Status(http.StatusNotFound)
		})

		It("restores an archived one, with archived_at null again", func() {
			var archived, restored views.ProductResponse
			app.Post(path+"/archive", nil).AsUser(staff).Do().Status(http.StatusOK).JSON(&archived)
			Expect(archived.ArchivedAt).NotTo(BeNil())
			app.Post(path+"/restore", nil).AsUser(staff).Do().Status(http.StatusOK).JSON(&restored)
			Expect(restored.ArchivedAt).To(BeNil())
		})

		It("refuses to delete one that was ordered, with 422", func() {
			user := factories.User(GinkgoT(), app.Store)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsUser(user).Do().Status(http.StatusCreated)
			app.Post("/orders", nil).AsUser(user).Do().Status(http.StatusCreated)

			app.Delete(path).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
		})

		It("pages the list by 20 and takes page, per_page and include_archived", func() {
			for i := 0; i < 21; i++ {
				factories.Product(GinkgoT(), app.Store)
			}
			var list views.ProductsResponse
			app.Get("/admin/products").AsUser(staff).Do().Status(http.StatusOK).JSON(&list)
			Expect(list.Products).To(HaveLen(20))
			Expect(list.Pagination.TotalPages).To(BeEquivalentTo(2))

			app.Get("/admin/products?page=2&per_page=10").AsUser(staff).Do().JSON(&list)
			Expect(list.Pagination.Page).To(BeEquivalentTo(2))
			app.Get("/admin/products?page=0").AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
		})

		It("is closed to customers, who cannot create or archive", func() {
			customer := factories.User(GinkgoT(), app.Store)
			app.Post("/admin/products", views.ProductRequest{Name: "Sneaky", PriceCents: 1, Stock: 1, ImageUrl: "/s"}).AsUser(customer).Do().Status(http.StatusForbidden)
			app.Post(path+"/archive", nil).AsUser(customer).Do().Status(http.StatusForbidden)

			var list views.ProductsResponse
			app.Get("/admin/products").AsUser(staff).Do().JSON(&list)
			Expect(list.Products).To(HaveLen(1))
			Expect(list.Products[0].ArchivedAt).To(BeNil())
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
