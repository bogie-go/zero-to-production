package admin_controller_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"

	"ecommerce/test/factories"
)

var _ = Describe("/admin", func() {
	It("is for staff only", func() {
		customer, staff := factories.User(GinkgoT(), app.Store), factories.Staff(GinkgoT(), app.Store)

		app.Get("/admin/products").Do().Status(http.StatusUnauthorized)
		app.Get("/admin/products").AsUser(customer).Do().Status(http.StatusForbidden)
		app.Get("/admin/products").AsGuest("someone").Do().Status(http.StatusForbidden)
		app.Get("/admin/products").AsUser(staff).Do().Status(http.StatusOK)
	})
})
