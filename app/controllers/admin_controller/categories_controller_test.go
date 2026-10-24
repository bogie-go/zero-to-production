package admin_controller_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("/admin/categories", func() {
	var staff domain.User

	BeforeEach(func() {
		staff = factories.Staff(GinkgoT(), app.Store)
	})

	It("lets staff create, rename and look up categories", func() {
		var created views.CategoryResponse
		app.Post("/admin/categories", views.CategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusCreated).JSON(&created)
		path := "/admin/categories/" + strconv.FormatInt(created.ID, 10)

		app.Post("/admin/categories", views.CategoryRequest{Name: "Audio"}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)
		app.Post("/admin/categories", views.CategoryRequest{Name: ""}).AsUser(staff).Do().Status(http.StatusUnprocessableEntity)

		var renamed views.CategoryResponse
		app.Patch(path, views.CategoryRequest{Name: "Sound"}).AsUser(staff).Do().Status(http.StatusOK).JSON(&renamed)
		Expect(renamed.Name).To(Equal("Sound"))
		app.Get("/admin/categories/999999").AsUser(staff).Do().Status(http.StatusNotFound)
	})

	It("is closed to customers, and to anyone without a token", func() {
		app.Post("/admin/categories", views.CategoryRequest{Name: "x"}).Do().Status(http.StatusUnauthorized)
		app.Post("/admin/categories", views.CategoryRequest{Name: "x"}).AsUser(factories.User(GinkgoT(), app.Store)).Do().Status(http.StatusForbidden)

		var list views.CategoriesResponse
		app.Get("/admin/categories").AsUser(staff).Do().JSON(&list)
		Expect(list.Categories).To(BeEmpty(), "nothing was created")
	})

	It("lists categories with their product counts", func() {
		category := factories.Category(GinkgoT(), app.Store)
		factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &category.ID })
		var list views.CategoriesResponse
		app.Get("/admin/categories").AsUser(staff).Do().Status(http.StatusOK).JSON(&list)
		Expect(list.Categories).To(HaveLen(1))
		Expect(list.Categories[0].ProductCount).To(BeEquivalentTo(1))
	})

	It("keeps a deleted category's products, just without a category", func() {
		category := factories.Category(GinkgoT(), app.Store)
		product := factories.Product(GinkgoT(), app.Store, func(p *domain.Product) { p.CategoryID = &category.ID })

		app.Delete("/admin/categories/" + strconv.FormatInt(category.ID, 10)).AsUser(staff).Do().Status(http.StatusNoContent)

		var shown views.ProductResponse
		app.Get("/products/" + strconv.FormatInt(product.ID, 10)).Do().Status(http.StatusOK).JSON(&shown)
		Expect(shown.CategoryID).To(BeNil())
		Expect(shown.CategoryName).To(BeEmpty())
	})
})
