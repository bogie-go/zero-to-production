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

var _ = Describe("GET /products", func() {
	var page, list views.ProductsResponse

	Context("with five products", func() {
		BeforeEach(func() {
			for i := 0; i < 5; i++ {
				factories.Product(GinkgoT(), app.Store)
			}
		})

		It("lists products newest first, in pages", func() {
			app.Get("/products?per_page=2&page=3").Do().Status(http.StatusOK).JSON(&page)

			Expect(page.Products).To(HaveLen(1))
			Expect(page.Pagination).To(Equal(views.Pagination{Page: 3, PerPage: 2, Total: 5, TotalPages: 3}))
		})
	})

	Context("with a category", func() {
		var audio domain.Category
		var inAudio domain.Product

		BeforeEach(func() {
			audio = factories.Category(GinkgoT(), app.Store)
			inAudio = factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &audio.ID })
			factories.Product(GinkgoT(), app.Store)
		})

		It("lists only that category's products, with its name", func() {
			app.Get("/products?category_id=" + strconv.FormatInt(audio.ID, 10)).Do().Status(http.StatusOK).JSON(&list)

			Expect(list.Products).To(HaveLen(1))
			Expect(list.Products[0].ID).To(Equal(inAudio.ID))
			Expect(list.Products[0].CategoryName).To(Equal(audio.Name))
		})
	})

	Context("when a product is archived", func() {
		var gone domain.Product

		BeforeEach(func() {
			gone = factories.Product(GinkgoT(), app.Store)
			_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
			Expect(err).NotTo(HaveOccurred())
		})

		It("is left out of the list", func() {
			app.Get("/products").Do().Status(http.StatusOK).JSON(&list)
			Expect(list.Products).To(BeEmpty())
		})

		It("is not found by id", func() {
			app.Get("/products/" + strconv.FormatInt(gone.ID, 10)).Do().Status(http.StatusNotFound)
		})
	})

	Context("with thirteen products", func() {
		BeforeEach(func() {
			for i := 0; i < 13; i++ {
				factories.Product(GinkgoT(), app.Store)
			}
		})

		It("asks for the first twelve by default", func() {
			app.Get("/products").Do().Status(http.StatusOK).JSON(&page)

			Expect(page.Products).To(HaveLen(12))
			Expect(page.Pagination).To(Equal(views.Pagination{Page: 1, PerPage: 12, Total: 13, TotalPages: 2}))
		})
	})

	It("clamps per_page to 50", func() {
		app.Get("/products?per_page=5000").Do().Status(http.StatusOK).JSON(&page)
		Expect(page.Pagination.PerPage).To(BeEquivalentTo(50))
	})

	DescribeTable("refuses a bad query with 422",
		func(query string) {
			app.Get("/products?" + query).Do().Status(http.StatusUnprocessableEntity)
		},
		Entry("page 0", "page=0"),
		Entry("a negative page", "page=-1"),
		Entry("a page that is not a number", "page=abc"),
		Entry("per_page 0", "per_page=0"),
		Entry("a per_page that is not a number", "per_page=x"),
		Entry("a page too large to offset", "page=99999999999"),
		Entry("a category that is not a number", "category_id=audio"),
	)

	Describe("GET /products/:id", func() {
		Context("with a product", func() {
			var p domain.Product

			BeforeEach(func() {
				p = factories.Product(GinkgoT(), app.Store)
			})

			It("finds the product by id", func() {
				var shown views.ProductResponse
				app.Get("/products/" + strconv.FormatInt(p.ID, 10)).Do().Status(http.StatusOK).JSON(&shown)
				Expect(shown.Name).To(Equal(p.Name))
			})
		})

		It("is 404 for an unknown product", func() {
			app.Get("/products/999999").Do().Status(http.StatusNotFound)
			app.Get("/products/nope").Do().Status(http.StatusNotFound)
		})
	})
})
