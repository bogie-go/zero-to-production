package admin_controllers_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"

	"ecommerce/app/models"
	"ecommerce/test/factories"
)

var _ = Describe("/admin", func() {
	var customer, staff models.User

	Context("with a customer and a staff account", func() {
		BeforeEach(func() {
			customer, staff = factories.User(GinkgoT(), app.Store), factories.Staff(GinkgoT(), app.Store)
		})

		It("answers 401 without a token", func() {
			app.Get("/admin/products").Do().Status(http.StatusUnauthorized)
		})

		It("is forbidden to a customer", func() {
			app.Get("/admin/products").AsUser(customer).Do().Status(http.StatusForbidden)
		})

		It("is forbidden to a guest", func() {
			app.Get("/admin/products").AsGuest("someone").Do().Status(http.StatusForbidden)
		})

		It("is open to staff", func() {
			app.Get("/admin/products").AsUser(staff).Do().Status(http.StatusOK)
		})
	})
})
