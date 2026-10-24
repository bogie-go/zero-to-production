package domain_test

import (
	"ecommerce/app/domain"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Category.Validate", func() {
	It("accepts a named category", func() {
		Expect(domain.Category{Name: "Audio"}.Validate()).To(Succeed())
	})

	DescribeTable("rejects a blank name",
		func(name string) {
			Expect(domain.Category{Name: name}.Validate()).To(MatchError(domain.ErrInvalid))
		},
		Entry("empty", ""),
		Entry("spaces only", "   "),
	)
})
