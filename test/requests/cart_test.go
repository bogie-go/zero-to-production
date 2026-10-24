package requests_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
	"ecommerce/test/testapp"
)

var _ = Describe("The cart", func() {

	Describe("a line", func() {
		var line string
		var cart views.CartResponse

		BeforeEach(func() {
			product := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.PriceCents = 300; p.Stock = 5 })
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("owner").Do().Status(http.StatusCreated).JSON(&cart)
			line = "/cart_items/" + strconv.FormatInt(cart.Items[0].ID, 10)
		})

		It("can be changed and removed by its owner", func() {
			app.Patch(line, views.UpdateCartItemRequest{Quantity: 4}).AsGuest("owner").Do().Status(http.StatusOK).JSON(&cart)
			Expect(cart.TotalCents).To(BeEquivalentTo(1200))

			app.Patch(line, views.UpdateCartItemRequest{Quantity: 6}).AsGuest("owner").Do().Status(http.StatusUnprocessableEntity).Contains("in stock")
			app.Patch(line, views.UpdateCartItemRequest{Quantity: 0}).AsGuest("owner").Do().Status(http.StatusUnprocessableEntity)

			app.Delete(line).AsGuest("owner").Do().Status(http.StatusOK).JSON(&cart)
			Expect(cart.Items).To(BeEmpty())
		})

		It("is not found through anyone else's cart", func() {
			app.Patch(line, views.UpdateCartItemRequest{Quantity: 2}).AsGuest("thief").Do().Status(http.StatusNotFound)
			app.Delete(line).AsGuest("thief").Do().Status(http.StatusNotFound)

			app.Get("/cart").AsGuest("owner").Do().JSON(&cart)
			Expect(cart.Items).To(HaveLen(1))
			Expect(cart.Items[0].Quantity).To(BeEquivalentTo(1))
		})
	})

	Describe("POST /cart_items", func() {
		var product, archived domain.Product

		BeforeEach(func() {
			product = factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.Stock = 2 })
			archived = factories.Product(GinkgoT(), app.Store)
			_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(archived.ID, 10))
			Expect(err).NotTo(HaveOccurred())
		})

		It("refuses bad lines", func() {
			add := func(productID int64, quantity int32) *testapp.Response {
				return app.Post("/cart_items", views.AddCartItemRequest{ProductID: productID, Quantity: quantity}).AsGuest("g").Do()
			}
			add(product.ID, 3).Status(http.StatusUnprocessableEntity)  // more than is in stock
			add(product.ID, 0).Status(http.StatusUnprocessableEntity)  // nothing
			add(999999, 1).Status(http.StatusUnprocessableEntity)      // no such product
			add(archived.ID, 1).Status(http.StatusUnprocessableEntity) // off the shop
			app.Post("/cart_items", "{").AsGuest("g").Do().Status(http.StatusBadRequest)
		})

		It("adds the same product twice to one line", func() {
			var cart views.CartResponse
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().Status(http.StatusCreated)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 1}).AsGuest("g").Do().JSON(&cart)

			Expect(cart.Items).To(HaveLen(1))
			Expect(cart.Items[0].Quantity).To(BeEquivalentTo(2))
		})
	})
})
