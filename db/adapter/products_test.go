package adapter_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("Products", func() {
	Describe("GetProduct", func() {
		DescribeTable("returns ErrNotFound",
			func(id string) {
				_, err := store.GetProduct(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("when the id is not a number", "nope"),
			Entry("when the id is empty", ""),
			Entry("when no product has that id", "999999999"),
		)

		Context("with a saved product", func() {
			var product models.Product

			BeforeEach(func() {
				product = factories.Product(GinkgoT(), store)
			})

			It("finds the product by id", func() {
				Expect(store.GetProduct(GinkgoT().Context(), strconv.FormatInt(product.ID, 10))).To(Equal(product))
			})
		})
	})

	Describe("ListProducts", func() {
		Context("with a saved product", func() {
			var product models.Product

			BeforeEach(func() {
				product = factories.Product(GinkgoT(), store)
			})

			It("includes the saved product", func() {
				all, err := store.ListProducts(GinkgoT().Context(), 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(HaveLen(1))
				Expect(all[0].ID).To(Equal(product.ID))
			})
		})
	})

	Describe("CreateProduct", func() {
		Context("with every field filled in", func() {
			It("saves the product with an id", func() {
				created, err := store.CreateProduct(GinkgoT().Context(), models.Product{
					Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 10, ImageUrl: "/widget.webp",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(created.ID).NotTo(BeZero())
			})
		})
	})

	Describe("UpdateProduct", func() {
		Context("with a saved product", func() {
			var product models.Product

			BeforeEach(func() {
				product = factories.Product(GinkgoT(), store)
			})

			It("stores the change", func() {
				product.Stock = 7
				updated, err := store.UpdateProduct(GinkgoT().Context(), product)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Stock).To(BeEquivalentTo(7))
			})
		})

		Context("when the product is not there", func() {
			It("is not found", func() {
				// A valid-looking product, so Validate passes and the
				// not-found check is what gets exercised.
				_, err := store.UpdateProduct(GinkgoT().Context(), models.Product{ID: 999999999, Name: "Nonexistent"})
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("DeleteProduct", func() {
		DescribeTable("returns ErrNotFound",
			func(id string) {
				Expect(store.DeleteProduct(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("when the id is not a number", "nope"),
			Entry("when the id is empty", ""),
			Entry("when no product has that id", "999999999"),
		)

		Context("with a saved product", func() {
			var product models.Product

			BeforeEach(func() {
				product = factories.Product(GinkgoT(), store)
			})

			It("removes the product", func() {
				id := strconv.FormatInt(product.ID, 10)
				Expect(store.DeleteProduct(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetProduct(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})
})
