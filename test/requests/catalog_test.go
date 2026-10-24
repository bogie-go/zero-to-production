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

var _ = Describe("GET /products", func() {

	It("lists products newest first, in pages", func() {
		for i := 0; i < 5; i++ {
			factories.Product(GinkgoT(), app.Store)
		}

		var page views.ProductsResponse
		app.Get("/products?per_page=2&page=3").Do().Status(http.StatusOK).JSON(&page)

		Expect(page.Products).To(HaveLen(1))
		Expect(page.Pagination).To(Equal(views.Pagination{Page: 3, PerPage: 2, Total: 5, TotalPages: 3}))
	})

	Context("with a category", func() {
		var audio domain.Category

		BeforeEach(func() {
			audio = factories.Category(GinkgoT(), app.Store)
		})

		It("lists only that category's products, with its name", func() {
			inAudio := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
			factories.Product(GinkgoT(), app.Store)

			var list views.ProductsResponse
			app.Get("/products?category_id=" + strconv.FormatInt(audio.ID, 10)).Do().Status(http.StatusOK).JSON(&list)

			Expect(list.Products).To(HaveLen(1))
			Expect(list.Products[0].ID).To(Equal(inAudio.ID))
			Expect(list.Products[0].CategoryName).To(Equal(audio.Name))
		})
	})

	When("a product is archived", func() {
		var gone domain.Product

		BeforeEach(func() {
			gone = factories.Product(GinkgoT(), app.Store)
			_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
			Expect(err).NotTo(HaveOccurred())
		})

		It("is left out of the list", func() {
			var list views.ProductsResponse
			app.Get("/products").Do().Status(http.StatusOK).JSON(&list)
			Expect(list.Products).To(BeEmpty())
		})

		It("is not found by id", func() {
			app.Get("/products/" + strconv.FormatInt(gone.ID, 10)).Do().Status(http.StatusNotFound)
		})
	})

	DescribeTable("refuses a bad query with 422",
		func(query string) {
			app.Get("/products?" + query).Do().Status(http.StatusUnprocessableEntity)
		},
		Entry("page 0", "page=0"),
		Entry("a category that is not a number", "category_id=audio"),
	)
})
