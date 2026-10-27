package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
)

var _ = Describe("Cart.Validate", func() {
	// A cart is identified by its token and has no other rules yet: whose
	// cart it is lives in the token, not in a column that could be wrong.
	Context("with a visitor's token", func() {
		var cart models.Cart

		BeforeEach(func() {
			cart = models.Cart{Token: "visitor-1"}
		})

		It("is valid", func() {
			Expect(cart.Validate()).To(Succeed())
		})
	})
})
