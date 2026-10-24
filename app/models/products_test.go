package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
)

var _ = Describe("Products", func() {

	DescribeTable("an unknown product is not found",
		func(id string) {
			_, err := store.GetProduct(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteProduct(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		// A valid-looking product, so Validate passes and the not-found check
		// is what gets exercised.
		_, err := store.UpdateProduct(GinkgoT().Context(), domain.Product{ID: 999999999, Name: "Nonexistent"})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	Describe("CreateProduct", func() {
		It("creates a product", func() {
			created, err := store.CreateProduct(GinkgoT().Context(), domain.Product{
				Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 10, ImageUrl: "/widget.webp",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeZero())
		})
	})

	Context("with a created product", func() {
		var created domain.Product
		var id string

		BeforeEach(func() {
			var err error
			created, err = store.CreateProduct(GinkgoT().Context(), domain.Product{
				Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 10, ImageUrl: "/widget.webp",
			})
			Expect(err).NotTo(HaveOccurred())
			id = strconv.FormatInt(created.ID, 10)
		})

		It("reads it", func() {
			Expect(store.GetProduct(GinkgoT().Context(), id)).To(Equal(created))
		})

		It("lists it", func() {
			all, err := store.ListProducts(GinkgoT().Context(), 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(all).To(HaveLen(1))
			Expect(all[0].ID).To(Equal(created.ID))
		})

		It("updates it", func() {
			created.Stock = 7
			updated, err := store.UpdateProduct(GinkgoT().Context(), created)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Stock).To(BeEquivalentTo(7))
		})

		It("deletes it", func() {
			Expect(store.DeleteProduct(GinkgoT().Context(), id)).To(Succeed())
			_, err := store.GetProduct(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
	})
})
