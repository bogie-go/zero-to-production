package controllers_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("GET /me", func() {
	It("is for accounts, not guests", func() {
		app.Get("/me").AsGuest("someone").Do().Status(http.StatusForbidden)
	})

	DescribeTable("protected endpoints need a token",
		func(path string) {
			app.Get(path).Do().Status(http.StatusUnauthorized)
			app.Get(path).WithAuthorization("Bearer garbage").Do().Status(http.StatusUnauthorized)
		},
		Entry("/me", "/me"),
		Entry("/cart", "/cart"),
		Entry("/orders", "/orders"),
	)
})
