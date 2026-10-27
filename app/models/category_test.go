package models_test

import (
	"ecommerce/app/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Category.Validate", func() {
	It("accepts a named category", func() {
		Expect(models.Category{Name: "Audio"}.Validate()).To(Succeed())
	})

	DescribeTable("rejects a blank name",
		func(name string) {
			Expect(models.Category{Name: name}.Validate()).To(MatchError(models.ErrInvalid))
		},
		Entry("empty", ""),
		Entry("spaces only", "   "),
	)
})
