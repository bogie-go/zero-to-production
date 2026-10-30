package admin_controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("/admin/categories", func() {
	var staff models.User

	Context("with a staff account", func() {
		BeforeEach(func() {
			staff = factories.Staff(GinkgoT(), app.Store)
		})

		Describe("POST /admin/categories", func() {
			It("lets staff create a category", func() {
				app.Post("/admin/categories", views.AdminCategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusCreated)
			})

			It("refuses an empty name", func() {
				app.Post("/admin/categories", views.AdminCategoryRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
			})
		})

		Context("with a category named Audio", func() {
			var path string

			BeforeEach(func() {
				var created views.AdminCategoryResponse
				app.Post("/admin/categories", views.AdminCategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
				path = "/admin/categories/" + strconv.FormatInt(created.ID, 10)
			})

			It("refuses a second category with the same name", func() {
				app.Post("/admin/categories", views.AdminCategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
			})

			It("lets staff rename it", func() {
				var renamed views.AdminCategoryResponse
				app.Patch(path, views.AdminCategoryRequest{Name: "Sound"}).AsUser(staff).Do().Status(http.StatusOK).JSON(&renamed)
				Expect(renamed.Name).To(Equal("Sound"))
			})
		})

		It("answers 404 for an unknown category", func() {
			app.Get("/admin/categories/999999").AsUser(staff).Do().Status(http.StatusNotFound)
		})

		Context("with a category holding a product", func() {
			var category models.Category
			var product models.Product

			BeforeEach(func() {
				category = factories.Category(GinkgoT(), app.Store)
				product = factories.Product(GinkgoT(), app.Store, func(p *models.Product) { p.CategoryID = &category.ID })
			})

			It("lists categories with their product counts", func() {
				var list views.AdminCategoriesResponse
				app.Get("/admin/categories").AsUser(staff).Do().Status(http.StatusOK).JSON(&list)
				Expect(list.Categories).To(HaveLen(1))
				Expect(list.Categories[0].ProductCount).To(BeEquivalentTo(1))
			})

			It("keeps a deleted category's products, just without a category", func() {
				app.Delete("/admin/categories/" + strconv.FormatInt(category.ID, 10)).AsUser(staff).Do().Status(http.StatusNoContent)

				var shown views.AdminProductResponse
				app.Get("/products/" + strconv.FormatInt(product.ID, 10)).Do().Status(http.StatusOK).JSON(&shown)
				Expect(shown.CategoryID).To(BeNil())
				Expect(shown.CategoryName).To(BeEmpty())
			})
		})

		Context("when a customer or nobody tries to create one", func() {
			var customer models.User

			BeforeEach(func() {
				customer = factories.User(GinkgoT(), app.Store)
			})

			It("answers 401 without a token and creates nothing", func() {
				app.Post("/admin/categories", views.AdminCategoryRequest{Name: "x"}).Do().Status(http.StatusUnauthorized)

				var list views.AdminCategoriesResponse
				app.Get("/admin/categories").AsUser(staff).Do().JSON(&list)
				Expect(list.Categories).To(BeEmpty(), "nothing was created")
			})

			It("answers 403 to a customer and creates nothing", func() {
				app.Post("/admin/categories", views.AdminCategoryRequest{Name: "x"}).AsUser(customer).Do().Status(http.StatusForbidden)

				var list views.AdminCategoriesResponse
				app.Get("/admin/categories").AsUser(staff).Do().JSON(&list)
				Expect(list.Categories).To(BeEmpty(), "nothing was created")
			})
		})
	})
})
