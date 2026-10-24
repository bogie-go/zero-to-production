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

	It("is created, read, listed, updated and deleted", func() {
		ctx := GinkgoT().Context()

		created, err := store.CreateProduct(ctx, domain.Product{
			Name: "Widget", Description: "a widget", PriceCents: 500, Stock: 10, ImageUrl: "/widget.webp",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(created.ID).NotTo(BeZero())
		id := strconv.FormatInt(created.ID, 10)

		Expect(store.GetProduct(ctx, id)).To(Equal(created))
		all, err := store.ListProducts(ctx, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(all).To(HaveLen(1))
		Expect(all[0].ID).To(Equal(created.ID))

		created.Stock = 7
		updated, err := store.UpdateProduct(ctx, created)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Stock).To(BeEquivalentTo(7))

		Expect(store.DeleteProduct(ctx, id)).To(Succeed())
		_, err = store.GetProduct(ctx, id)
		Expect(err).To(MatchError(domain.ErrNotFound))
	})
})
