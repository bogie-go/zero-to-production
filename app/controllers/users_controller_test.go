package controllers_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("POST /users", func() {
	creds := views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}

	It("signs up", func() {
		app.Post("/users", creds).Do().Status(http.StatusCreated)
	})

	Context("with an account already signed up", func() {
		BeforeEach(func() {
			app.Post("/users", creds).Do().Status(http.StatusCreated)
		})

		It("logs in", func() {
			var login views.SessionResponse
			app.Post("/sessions", views.LoginRequest(creds)).Do().Status(http.StatusOK).JSON(&login)

			Expect(login.Token).NotTo(BeEmpty())
			Expect(app.Jobs("send_sign_in_alert")).To(Equal(1))
		})

		It("says who you are once logged in", func() {
			var login views.SessionResponse
			app.Post("/sessions", views.LoginRequest(creds)).Do().Status(http.StatusOK).JSON(&login)

			var me views.MeResponse
			app.Get("/me").WithAuthorization("Bearer " + login.Token).Do().Status(http.StatusOK).JSON(&me)
			Expect(me.Email).To(Equal("ada@example.com"))
			Expect(me.Role).To(Equal(domain.RoleCustomer))
		})

		It("refuses a taken email", func() {
			app.Post("/users", creds).Do().Status(http.StatusUnprocessableEntity).Contains("already")
		})

		DescribeTable("refuses bad input",
			func(body any, status int) {
				app.Post("/users", body).Do().Status(status)
			},
			Entry("a malformed email", views.SignupRequest{Email: "not-an-email", Password: "correct-horse-battery"}, http.StatusUnprocessableEntity),
			Entry("a short password", views.SignupRequest{Email: "grace@example.com", Password: "short"}, http.StatusUnprocessableEntity),
			Entry("a body that is not JSON", "{not json", http.StatusBadRequest),
		)

		It("never lets the request choose the role", func() {
			var session views.SessionResponse
			app.Post("/users", `{"email":"mallory@example.com","password":"correct-horse-battery","role":"staff"}`).
				Do().Status(http.StatusCreated).JSON(&session)

			app.Get("/admin/products").WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusForbidden)
		})
	})

	It("still creates the account with a bad guest token: a cart that cannot be merged is not a reason to refuse", func() {
		app.Post("/users", creds).WithAuthorization("Bearer not-a-real-token").Do().Status(http.StatusCreated)
	})

	Context("with a guest who filled a cart with two of a product", func() {
		var guest views.GuestSessionResponse

		BeforeEach(func() {
			product := factories.Product(GinkgoT(), app.Store)
			app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).
				WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)
		})

		It("merges the guest's cart into the new account", func() {
			var session views.SessionResponse
			app.Post("/users", creds).WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated).JSON(&session)

			var cart views.CartResponse
			app.Get("/cart").WithAuthorization("Bearer " + session.Token).Do().Status(http.StatusOK).JSON(&cart)
			Expect(cart.Items).To(HaveLen(1))
			Expect(cart.Items[0].Quantity).To(BeEquivalentTo(2))
		})
	})
})
