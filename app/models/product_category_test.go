package models_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("ProductCategory.Validate", func() {
	var productCategory models.ProductCategory

	BeforeEach(func() {
		productCategory = factories.NewProductCategory()
	})

	Context("with the factory's defaults", func() {
		It("is valid", func() {
			Expect(productCategory.Validate()).To(Succeed())
		})
	})

	// Add a Context for each rule you put in Validate.
})
