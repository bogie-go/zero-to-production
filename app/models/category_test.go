package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("Category.Validate", func() {
	var category models.Category

	BeforeEach(func() {
		category = models.Category{Name: "Audio"}
	})

	Context("with a name", func() {
		It("is valid", func() {
			Expect(category.Validate()).To(Succeed())
		})
	})

	DescribeTable("is refused for",
		func(name string) {
			category.Name = name

			Expect(category.Validate()).To(MatchError(models.ErrInvalid))
		},
		Entry("an empty name", ""),
		Entry("a name of spaces", "   "),
	)
})
