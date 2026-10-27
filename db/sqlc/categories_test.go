package db_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("Categories", func() {
	Describe("GetCategory", func() {
		DescribeTable("is not found for",
			func(id string) {
				_, err := store.GetCategory(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved category", func() {
			var category models.Category

			BeforeEach(func() {
				category = factories.Category(GinkgoT(), store)
			})

			It("finds the category by id", func() {
				Expect(store.GetCategory(GinkgoT().Context(), strconv.FormatInt(category.ID, 10))).To(Equal(category))
			})
		})
	})

	Describe("CreateCategory", func() {
		Context("with a name", func() {
			It("saves the category with an id", func() {
				created, err := store.CreateCategory(GinkgoT().Context(), models.Category{Name: "Audio"})
				Expect(err).NotTo(HaveOccurred())
				Expect(created.ID).NotTo(BeZero())
			})
		})

		Context("with a blank name", func() {
			It("is refused", func() {
				_, err := store.CreateCategory(GinkgoT().Context(), models.Category{Name: "  "})
				Expect(err).To(MatchError(models.ErrInvalid))
			})
		})

		Context("with a name another category already has", func() {
			var existing models.Category

			BeforeEach(func() {
				existing = factories.Category(GinkgoT(), store)
			})

			It("is refused", func() {
				_, err := store.CreateCategory(GinkgoT().Context(), models.Category{Name: existing.Name})
				Expect(err).To(MatchError(models.ErrInvalid))
			})
		})
	})

	Describe("UpdateCategory", func() {
		Context("with a saved category", func() {
			var category models.Category

			BeforeEach(func() {
				category = factories.Category(GinkgoT(), store)
			})

			It("stores the new name", func() {
				category.Name = "Sound"
				updated, err := store.UpdateCategory(GinkgoT().Context(), category)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Name).To(Equal("Sound"))
			})
		})

		Context("when the category is not there", func() {
			It("is not found", func() {
				_, err := store.UpdateCategory(GinkgoT().Context(), models.Category{ID: 999999999, Name: "Nope"})
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})

	Describe("DeleteCategory", func() {
		DescribeTable("is not found for",
			func(id string) {
				Expect(store.DeleteCategory(GinkgoT().Context(), id)).To(MatchError(models.ErrNotFound))
			},
			Entry("a malformed id", "nope"),
			Entry("an empty id", ""),
			Entry("a well-formed id that names nothing", "999999999"),
		)

		Context("with a saved category", func() {
			var category models.Category

			BeforeEach(func() {
				category = factories.Category(GinkgoT(), store)
			})

			It("removes the category", func() {
				id := strconv.FormatInt(category.ID, 10)
				Expect(store.DeleteCategory(GinkgoT().Context(), id)).To(Succeed())
				_, err := store.GetCategory(GinkgoT().Context(), id)
				Expect(err).To(MatchError(models.ErrNotFound))
			})
		})
	})
})
