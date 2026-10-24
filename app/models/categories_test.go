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

	Describe("CreateCategory", func() {
		It("creates a category", func() {
			created, err := store.CreateCategory(GinkgoT().Context(), domain.Category{Name: "Audio"})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.ID).NotTo(BeZero())
		})
	})

	Context("with a created category", func() {
		var created domain.Category
		var id string

		BeforeEach(func() {
			var err error
			created, err = store.CreateCategory(GinkgoT().Context(), domain.Category{Name: "Audio"})
			Expect(err).NotTo(HaveOccurred())
			id = strconv.FormatInt(created.ID, 10)
		})

		It("reads it", func() {
			Expect(store.GetCategory(GinkgoT().Context(), id)).To(Equal(created))
		})

		It("renames it", func() {
			created.Name = "Sound"
			updated, err := store.UpdateCategory(GinkgoT().Context(), created)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("Sound"))
		})

		It("deletes it", func() {
			Expect(store.DeleteCategory(GinkgoT().Context(), id)).To(Succeed())
			_, err := store.GetCategory(GinkgoT().Context(), id)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("refuses a second category with the same name", func() {
			_, err := store.CreateCategory(GinkgoT().Context(), domain.Category{Name: "Audio"})
			Expect(err).To(MatchError(domain.ErrInvalid))
		})
	})

	Describe("the name", func() {
		It("is required", func() {
			_, err := store.CreateCategory(GinkgoT().Context(), domain.Category{Name: "  "})
			Expect(err).To(MatchError(domain.ErrInvalid))
		})
	})
})
