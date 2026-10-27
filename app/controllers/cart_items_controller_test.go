package controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testapp"
)

var _ = Describe("Cart items", func() {

	Describe("a line", func() {
		var line string
		var cart views.CartResponse

		Context("in a guest's cart", func() {
			BeforeEach(func() {
				product := factories.Product(GinkgoT(), app.Store, func(p *models.Product) { p.PriceCents = 300; p.Stock = 5 })
				app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("owner").Do().Status(http.StatusCreated).JSON(&cart)
				line = "/cart_items/" + strconv.FormatInt(cart.Items[0].ID, 10)
			})

			It("can be changed by its owner", func() {
				app.Patch(line, views.UpdateCartItemRequest{Quantity: 4}).AsGuest("owner").Do().Status(http.StatusOK).JSON(&cart)
				Expect(cart.TotalCents).To(BeEquivalentTo(1200))
			})

			It("refuses a quantity beyond the stock", func() {
				app.Patch(line, views.UpdateCartItemRequest{Quantity: 6}).AsGuest("owner").Do().Status(http.StatusUnprocessableEntity).Contains("in stock")
			})

			It("refuses a quantity of zero", func() {
				app.Patch(line, views.UpdateCartItemRequest{Quantity: 0}).AsGuest("owner").Do().Status(http.StatusUnprocessableEntity)
			})

			It("can be removed by its owner", func() {
				app.Delete(line).AsGuest("owner").Do().Status(http.StatusOK).JSON(&cart)
				Expect(cart.Items).To(BeEmpty())
			})

			It("is 404 when the line does not exist", func() {
				app.Patch("/cart_items/999999", views.UpdateCartItemRequest{Quantity: 2}).AsGuest("owner").Do().Status(http.StatusNotFound)
				app.Delete("/cart_items/999999").AsGuest("owner").Do().Status(http.StatusNotFound)
			})

			Context("when someone else tries to change it", func() {
				BeforeEach(func() {
					app.Patch(line, views.UpdateCartItemRequest{Quantity: 2}).AsGuest("thief").Do().Status(http.StatusNotFound)
					app.Delete(line).AsGuest("thief").Do().Status(http.StatusNotFound)
				})

				It("leaves the owner's cart untouched", func() {
					app.Get("/cart").AsGuest("owner").Do().JSON(&cart)
					Expect(cart.Items).To(HaveLen(1))
					Expect(cart.Items[0].Quantity).To(BeEquivalentTo(1))
				})
			})
		})
	})

	Describe("POST /cart_items", func() {
		var product, archived models.Product

		Context("with a product with two in stock and an archived product", func() {
			BeforeEach(func() {
				product = factories.Product(GinkgoT(), app.Store, func(p *models.Product) { p.Stock = 2 })
				archived = factories.Product(GinkgoT(), app.Store)
				_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(archived.ID, 10))
				Expect(err).NotTo(HaveOccurred())
			})

			Describe("a bad line", func() {
				add := func(productID int64, quantity int32) *testapp.Response {
					return app.Post("/cart_items", views.AddCartItemRequest{ProductID: productID, Quantity: quantity}).AsGuest("g").Do()
				}

				It("is refused when it is more than is in stock", func() {
					add(product.ID, 3).Status(http.StatusUnprocessableEntity)
				})

				It("is refused when it is nothing", func() {
					add(product.ID, 0).Status(http.StatusUnprocessableEntity)
				})

				It("is refused when there is no such product", func() {
					add(999999, 1).Status(http.StatusUnprocessableEntity)
				})

				It("is refused when the product is off the shop", func() {
					add(archived.ID, 1).Status(http.StatusUnprocessableEntity)
				})

				It("is refused with 400 when the body is malformed", func() {
					app.Post("/cart_items", "{").AsGuest("g").Do().Status(http.StatusBadRequest)
				})
			})

			Context("when the product is already in the cart", func() {
				BeforeEach(func() {
					app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().Status(http.StatusCreated)
				})

				It("adds the same product to one line", func() {
					var cart views.CartResponse
					app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().JSON(&cart)

					Expect(cart.Items).To(HaveLen(1))
					Expect(cart.Items[0].Quantity).To(BeEquivalentTo(2))
				})
			})
		})
	})
})
