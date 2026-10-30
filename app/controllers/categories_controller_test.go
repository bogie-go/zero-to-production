package controllers_test

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("GET /categories", func() {
	It("is an empty list, not null, with no categories", func() {
		app.Get("/categories").Do().Status(http.StatusOK).Contains(`"categories":[]`)
	})

	Context("with a category holding a live and an archived product, and an empty one", func() {
		var audio, empty models.Category

		BeforeEach(func() {
			audio, empty = factories.Category(GinkgoT(), app.Store), factories.Category(GinkgoT(), app.Store)
			factories.Categorize(GinkgoT(), app.Store, factories.Product(GinkgoT(), app.Store), audio)
			gone := factories.Categorize(GinkgoT(), app.Store, factories.Product(GinkgoT(), app.Store), audio)
			_, err := app.Store.ArchiveProduct(GinkgoT().Context(), strconv.FormatInt(gone.ID, 10))
			Expect(err).NotTo(HaveOccurred())
		})

		It("counts only live products and lists empty categories with 0", func() {
			var list views.CategoriesResponse
			app.Get("/categories").Do().Status(http.StatusOK).JSON(&list)

			counts := map[int64]int64{}
			for _, c := range list.Categories {
				counts[c.ID] = c.ProductCount
			}
			Expect(counts).To(Equal(map[int64]int64{audio.ID: 1, empty.ID: 0}))
		})
	})
})
