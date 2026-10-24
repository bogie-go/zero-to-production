package requests_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/domain"
	"ecommerce/app/views"
	"ecommerce/test/factories"
)

var _ = Describe("Accounts", func() {
	creds := views.SignupRequest{Email: "ada@example.com", Password: "correct-horse-battery"}

	It("signs up, logs in and says who you are", func() {
		app.Post("/users", creds).Do().Status(http.StatusCreated)

		var login views.SessionResponse
		app.Post("/sessions", views.LoginRequest(creds)).Do().Status(http.StatusOK).JSON(&login)
		Expect(login.Token).NotTo(BeEmpty())
		Expect(app.Jobs("send_sign_in_alert")).To(Equal(1))

		var me views.MeResponse
		app.Get("/me").WithAuthorization("Bearer " + login.Token).Do().Status(http.StatusOK).JSON(&me)
		Expect(me.Email).To(Equal("ada@example.com"))
		Expect(me.Role).To(Equal(domain.RoleCustomer))
	})

	Describe("POST /users", func() {
		BeforeEach(func() {
			app.Post("/users", creds).Do().Status(http.StatusCreated)
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

	Describe("POST /sessions", func() {
		BeforeEach(func() {
			app.Post("/users", creds).Do().Status(http.StatusCreated)
		})

		It("does not say which part was wrong", func() {
			wrong := app.Post("/sessions", views.LoginRequest{Email: "ada@example.com", Password: "wrong-password-here"}).Do().Status(http.StatusUnauthorized)
			unknown := app.Post("/sessions", views.LoginRequest{Email: "nobody@example.com", Password: "correct-horse-battery"}).Do().Status(http.StatusUnauthorized)

			Expect(wrong.Body).To(Equal(unknown.Body))
			Expect(app.Jobs("send_sign_in_alert")).To(BeZero())
		})

		It("keeps the cart a guest filled", func() {
			product := factories.Product(GinkgoT(), app.Store)
			var guest views.GuestSessionResponse
			app.Post("/guest_sessions", nil).Do().Status(http.StatusCreated).JSON(&guest)
			app.Post("/cart_items", views.AddCartItemRequest{ProductID: product.ID, Quantity: 2}).
				WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusCreated)

			var login views.SessionResponse
			app.Post("/sessions", views.LoginRequest(creds)).
				WithAuthorization("Bearer " + guest.Token).Do().Status(http.StatusOK).JSON(&login)

			var cart views.CartResponse
			app.Get("/cart").WithAuthorization("Bearer " + login.Token).Do().JSON(&cart)
			Expect(cart.Items).To(HaveLen(1))
			Expect(cart.Items[0].Quantity).To(BeEquivalentTo(2))
		})
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

	It("needs a token to check out", func() {
		app.Post("/orders", nil).Do().Status(http.StatusUnauthorized)
	})
})
