package models_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
)

var _ = Describe("Categories", func() {

	DescribeTable("an unknown category is not found",
		func(id string) {
			_, err := store.GetCategory(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(store.DeleteCategory(GinkgoT().Context(), id)).To(MatchError(domain.ErrNotFound))
		},
		Entry("a malformed id", "nope"),
		Entry("an empty id", ""),
		Entry("a well-formed id that names nothing", "999999999"),
	)

	It("cannot update one that is not there", func() {
		_, err := store.UpdateCategory(GinkgoT().Context(), domain.Category{ID: 999999999, Name: "Nope"})
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	It("is created, read, renamed and deleted", func() {
		ctx := GinkgoT().Context()

		created, err := store.CreateCategory(ctx, domain.Category{Name: "Audio"})
		Expect(err).NotTo(HaveOccurred())
		id := strconv.FormatInt(created.ID, 10)
		Expect(store.GetCategory(ctx, id)).To(Equal(created))

		created.Name = "Sound"
		updated, err := store.UpdateCategory(ctx, created)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Name).To(Equal("Sound"))

		Expect(store.DeleteCategory(ctx, id)).To(Succeed())
		_, err = store.GetCategory(ctx, id)
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	Describe("the name", func() {
		It("is required", func() {
			_, err := store.CreateCategory(GinkgoT().Context(), domain.Category{Name: "  "})
			Expect(err).To(MatchError(domain.ErrInvalid))
		})

		It("is unique", func() {
			_, err := store.CreateCategory(GinkgoT().Context(), domain.Category{Name: "Audio"})
			Expect(err).NotTo(HaveOccurred())
			_, err = store.CreateCategory(GinkgoT().Context(), domain.Category{Name: "Audio"})
			Expect(err).To(MatchError(domain.ErrInvalid))
		})
	})
})
