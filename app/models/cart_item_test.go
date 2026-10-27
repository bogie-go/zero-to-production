package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("CartItem.Validate", func() {
	var item models.CartItem

	BeforeEach(func() {
		item = models.CartItem{CartID: 1, ProductID: 1, Quantity: 1}
	})

	Context("with a quantity of one", func() {
		It("is valid", func() {
			Expect(item.Validate()).To(Succeed())
		})
	})

	DescribeTable("is refused for",
		func(quantity int32) {
			item.Quantity = quantity

			err := item.Validate()

			Expect(err).To(MatchError(models.ErrInvalid))
			Expect(err).To(MatchError(ContainSubstring("quantity must be positive")))
		},
		Entry("a quantity of zero", int32(0)),
		Entry("a negative quantity", int32(-3)),
	)
})
