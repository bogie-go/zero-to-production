package adapter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("ProductCategories", func() {
	Describe("GetProductCategory", func() {
		DescribeTable("returns ErrNotFound",
			func(id string) {
				_, err := store.GetProductCategory(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("when the id is not a number", "nope"),
			Entry("when the id is empty", ""),
			Entry("when no product category has that id", "999999999"),
		)
	})

	Describe("DeleteProductCategory", func() {
		DescribeTable("returns ErrNotFound",
			func(id string) {
				Expect(store.DeleteProductCategory(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("when the id is not a number", "nope"),
			Entry("when the id is empty", ""),
			Entry("when no product category has that id", "999999999"),
		)
	})

	Describe("UpdateProductCategory", func() {
		Context("when the product category is not there", func() {
			var missing models.ProductCategory

			BeforeEach(func() {
				missing = factories.NewProductCategory()
				missing.ID = 999999999
			})

			It("is not found", func() {
				_, err := store.UpdateProductCategory(GinkgoT().Context(), missing)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("CreateProductCategory", func() {
		Context("when a reference names nothing", func() {
			var missing models.ProductCategory

			BeforeEach(func() {
				missing = factories.NewProductCategory()
				missing.ProductID = 999999999
				missing.CategoryID = 999999999
			})

			It("is invalid", func() {
				_, err := store.CreateProductCategory(GinkgoT().Context(), missing)
				Expect(err).To(MatchError(models.ErrInvalid))
			})
		})
	})

	// No round trip here: a product category belongs to a parent that has to be saved
	// first. Make the parent with its factory and add the specs, with the
	// reference set in the factory's "with" function.
})
